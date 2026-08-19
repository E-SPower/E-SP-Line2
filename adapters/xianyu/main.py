"""
E-SP-Line2 XianYu (闲鱼) Adapter 主入口
=========================================

直接复用 vendor/ 中的 XianYuApis 逆向 SDK(XianyuLive / XianyuApis)，
并通过 ESPL 桥接层(esp_bridge.py)与后端 WebSocket 通信。

多开多配置：
- 每个实例一个独立的 Cookie/device_id 配置(通过后端 WebUI 实例管理填写)
- 可同时运行多个实例，每个实例独立连接闲鱼 WS + 后端 WS

用法：
    # 单实例
    python main.py --instance-id <INSTANCE_ID> --backend http://localhost:8080

    # 多实例(逗号分隔)
    python main.py --instance-id aaa,bbb,ccc --backend http://localhost:8080

    # 指定后端 JWT token
    python main.py --instance-id aaa --backend http://localhost:8080 --token <JWT>
"""

from __future__ import annotations

import argparse
import asyncio
import os
import sys
from typing import List

# 将 vendor 目录加入导入路径，直接复用 XianYuApis 原始 SDK
_VENDOR_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "vendor")
if _VENDOR_DIR not in sys.path:
    sys.path.insert(0, _VENDOR_DIR)

from loguru import logger

logger.remove()
logger.add(
    sink=lambda msg: print(msg, end="", flush=True),
    format="<green>{time:YYYY-MM-DD HH:mm:ss}</green> | <level>{level: <8}</level> | <level>{message}</level>",
    colorize=False,
)

from esp_bridge import EspBridge, fetch_instance_config


def parse_args(argv: List[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="XianYu adapter for E-SP-Line2")
    parser.add_argument(
        "--instance-id",
        required=True,
        help="Adapter instance ID(s), comma separated for multi-instance",
    )
    parser.add_argument(
        "--backend",
        default=os.environ.get("ESP_BACKEND_URL", "http://localhost:8080"),
        help="E-SP-Line2 backend URL",
    )
    parser.add_argument(
        "--token",
        default=os.environ.get("ESP_TOKEN", ""),
        help="Backend JWT token (optional)",
    )
    return parser.parse_args(argv)


class XianyuInstance:
    """
    单个闲鱼实例的完整生命周期：
    - 从后端拉取配置(cookie / device_id)
    - 创建 vendor 的 XianyuLive(复用完整逆向逻辑)
    - 创建 ESPL 桥接器(连接后端 WS)
    - 闲鱼消息 -> 桥接上报后端；后端指令 -> 闲鱼发送
    """

    def __init__(self, backend_url: str, instance_id: str, token: str = ""):
        self.backend_url = backend_url
        self.instance_id = instance_id
        self.token = token
        self.cfg = None
        self.live = None
        self.bridge: EspBridge | None = None

    async def start(self):
        # 1. 从后端拉取实例配置(WebUI 中填写的 cookie / device_id)
        logger.info(f"[{self.instance_id}] ===== 启动闲鱼实例 =====")
        self.cfg = fetch_instance_config(self.backend_url, self.instance_id, self.token)
        cookie = self.cfg.config.get("cookie", "")
        if not cookie:
            logger.error(
                f"[{self.instance_id}] 实例未配置 Cookie，无法启动。"
                "请在 WebUI 实例管理中填写闲鱼 Cookie。"
            )
            raise RuntimeError(f"Instance {self.instance_id} has no cookie configured")

        logger.info(
            f"[{self.instance_id}] 实例信息: 名称={self.cfg.name} "
            f"接入器={self.cfg.adapter_id} 平台={self.cfg.platform_id} "
            f"has_cookie=True"
        )

        # 2. 创建 ESPL 桥接器(连接后端，接收指令)
        self.bridge = EspBridge(
            backend_url=self.backend_url,
            instance_id=self.instance_id,
            token=self.token,
            reconnect_delay=self.cfg.config.get("reconnect_delay", 5),
            on_inbound=self._handle_backend_command,
        )

        # 3. 创建 vendor 的 XianyuLive(复用完整逆向逻辑)
        from goofish_live import XianyuLive

        device_id = self.cfg.config.get("device_id", "")
        self.live = XianyuLive(
            cookies_str=cookie,
            message_callback=self._handle_xianyu_message,
            device_id_override=device_id or None,
        )
        # 绑定桥接器，供兜底上报与指令发送使用
        self.live.bind_bridge(self.instance_id, self.bridge)
        logger.info(
            f"[{self.instance_id}] 闲鱼客户端已创建: device_id={'******' if device_id else '自动生成'}"
        )

        # 4. 并行运行：后端连接 + 闲鱼监听
        logger.info(f"[{self.instance_id}] 开始运行: 连接后端 + 监听闲鱼消息")
        await asyncio.gather(
            self.bridge.connect_forever(),
            self.live.main(),
        )

    async def _handle_xianyu_message(self, websocket, cid, send_user_id, send_user_name, send_message,
                                     raw_message=None, message_chain=None):
        """闲鱼收到消息 -> 通过桥接上报后端。

        现在上报包含完整的原始消息(raw)和消息链(message_chain)，
        确保后端保存时不会丢失任何信息。
        """
        if not self.bridge:
            return
        payload = {
            "platform_id": "xianyu",
            "conversation_id": cid,
            "sender_id": send_user_id,
            "sender_name": send_user_name,
            "message_type": "text",
            "message_content": send_message,
            "idempotency_key": f"xianyu-{send_user_id}-{cid}-{int(__import__('time').time()*1000)}",
        }
        # 保存完整的原始平台消息
        if raw_message is not None:
            payload["raw"] = raw_message
        # 保存消息链
        if message_chain is not None:
            payload["message_chain"] = message_chain
        await self.bridge.send_inbound(payload)
        logger.info(f"[{self.instance_id}] Reported inbound message to backend")

    async def _handle_backend_command(self, data: dict):
        """后端下发指令 -> 调用闲鱼发送。"""
        if not self.live:
            return
        command_type = data.get("command_type") or data.get("type")
        payload = data.get("payload", {})
        if not payload or not self.live.ws:
            logger.warning(f"[{self.instance_id}] No payload or not connected")
            return

        if command_type in ("send_text", "send"):
            text = payload.get("text") or payload.get("message_content", "")
            await self.live.send_msg(self.live.ws, payload["cid"], payload["toid"], {"type": "text", "text": text})
        elif command_type == "send_image":
            await self.live.send_msg(
                self.live.ws,
                payload["cid"],
                payload["toid"],
                {
                    "type": "image",
                    "image_url": payload.get("image_url", ""),
                    "width": payload.get("width", 0),
                    "height": payload.get("height", 0),
                },
            )
        else:
            logger.warning(f"[{self.instance_id}] Unknown command: {command_type}")


# ASCII art banner, same as the E-SP-Line2 backend startup banner.
BANNER = r"""
  ███████╗   ███████╗██████╗     ██╗     ██╗███╗   ██╗███████╗██████╗
  ██╔════╝   ██╔════╝██╔══██╗    ██║     ██║████╗  ██║██╔════╝╚════██╗
  █████╗     ███████╗██████╔╝    ██║     ██║██╔██╗ ██║█████╗    ▄███╔╝
  ██╔══╝     ╚════██║██╔═══╝     ██║     ██║██║╚██╗██║██╔══╝  ▄▀══╝
  ███████╗   ███████║██║         ███████╗██║██║ ╚████║███████╗███████╗
  ╚══════╝   ╚══════╝╚═╝         ╚══════╝╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝

  Power By LangBot-community-team

  --------------------------------------------------------------------
"""


async def main(argv: List[str]) -> int:
    args = parse_args(argv)
    instance_ids = [i.strip() for i in args.instance_id.split(",") if i.strip()]
    if not instance_ids:
        logger.error("No instance IDs provided")
        return 1

    print(BANNER, flush=True)
    logger.info(f"Starting XianYu adapter for instances: {instance_ids}")
    instances = [XianyuInstance(args.backend, i, args.token) for i in instance_ids]

    # 并发启动所有实例(多开)
    tasks = [asyncio.create_task(inst.start()) for inst in instances]

    try:
        await asyncio.gather(*tasks)
    except KeyboardInterrupt:
        logger.info("Shutting down...")
        await shutdown(instances)
    except Exception as e:
        logger.error(f"适配器异常退出: {e}")
        await shutdown(instances)
        return 1
    return 0


async def shutdown(instances) -> None:
    for inst in instances:
        try:
            if inst.live:
                await inst.live.close()
        except Exception:
            pass
        try:
            if inst.bridge:
                await inst.bridge.close()
        except Exception:
            pass


if __name__ == "__main__":
    sys.exit(asyncio.run(main(sys.argv[1:])))

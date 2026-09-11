"""
E-SP-Line2 Taobao (淘宝) Adapter 主入口
=========================================

直接复用 vendor/ 中的 TaobaoApis 逆向 SDK(taobaoLive / TaobaoApis)，
并通过 ESPL 桥接层(esp_bridge.py)与后端 WebSocket 通信。

多开多配置：
- 每个实例一个独立的 Cookie/device_id 配置(通过后端 WebUI 实例管理填写)
- 可同时运行多个实例，每个实例独立连接淘宝 WS + 后端 WS

用法：
    # 单实例
    python main.py --instance-id <INSTANCE_ID> --backend http://localhost:8080

    # 多实例(逗号分隔)
    python main.py --instance-id aaa,bbb,ccc --backend http://localhost:8080

    # 指定后端 JWT token
    python main.py --instance-id aaa --backend http://localhost:8080 --token <JWT>
"""

from __future__ import annotations

# loguru / esp_bridge 需要在把 vendor 目录加入 sys.path 之后再导入，
# 因此无法全部置于文件顶部；taobao_live 更是延迟到运行时导入。
# 这些依赖安装在系统解释器下，静态分析器未必能解析，故一并豁免。
# pylint: disable=wrong-import-position,import-error,import-outside-toplevel

import argparse
import asyncio
import os
import sys
import time
from typing import List

# 将 vendor 目录加入导入路径，直接复用 TaobaoApis 原始 SDK
_VENDOR_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "vendor")
if _VENDOR_DIR not in sys.path:
    sys.path.insert(0, _VENDOR_DIR)

from loguru import logger

_LOG_FORMAT = (
    "<green>{time:YYYY-MM-DD HH:mm:ss}</green> | "
    "<level>{level: <8}</level> | <level>{message}</level>"
)

logger.remove()
logger.add(
    sink=lambda msg: print(msg, end=""),
    format=_LOG_FORMAT,
    colorize=False,
)

from esp_bridge import EspBridge, fetch_instance_config


def parse_args(argv: List[str]) -> argparse.Namespace:
    """解析命令行参数（--instance-id / --backend / --token）。"""
    parser = argparse.ArgumentParser(description="Taobao adapter for E-SP-Line2")
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


class TaobaoInstance:
    """
    单个淘宝实例的完整生命周期：
    - 从后端拉取配置(cookie / device_id)
    - 创建 vendor 的 taobaoLive(复用完整逆向逻辑)
    - 创建 ESPL 桥接器(连接后端 WS)
    - 淘宝消息 -> 桥接上报后端；后端指令 -> 淘宝发送
    """

    def __init__(self, backend_url: str, instance_id: str, token: str = ""):
        """初始化单个淘宝实例。

        Args:
            backend_url: E-SP-Line2 后端地址。
            instance_id: 当前适配器实例 ID。
            token: 可选的后端 JWT token。
        """
        self.backend_url = backend_url
        self.instance_id = instance_id
        self.token = token
        self.cfg = None
        self.live = None
        self.bridge: EspBridge | None = None

    async def start(self):
        """启动实例：拉取配置、创建桥接器与淘宝客户端，并并发运行。"""
        # 1. 从后端拉取实例配置(WebUI 中填写的 cookie / device_id)
        logger.info(f"[{self.instance_id}] ===== 启动淘宝实例 =====")
        self.cfg = fetch_instance_config(self.backend_url, self.instance_id, self.token)
        cookie = self.cfg.config.get("cookie", "")
        if not cookie:
            logger.error(
                f"[{self.instance_id}] 实例未配置 Cookie，无法启动。"
                "请在 WebUI 实例管理中填写淘宝 Cookie。"
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

        # 3. 创建 vendor 的 taobaoLive(复用完整逆向逻辑)
        # 延迟导入：taobao_live 位于运行时才加入 sys.path 的 vendor 目录。
        from taobao_live import taobaoLive  # pylint: disable=import-outside-toplevel

        device_id = self.cfg.config.get("device_id", "")
        self.live = taobaoLive(
            cookies_str=cookie,
            message_callback=self._handle_taobao_message,
            device_id_override=device_id or None,
        )
        # 绑定桥接器，供兜底上报与指令发送使用
        self.live.bind_bridge(self.instance_id, self.bridge)
        device_id_desc = "******" if device_id else "自动生成"
        logger.info(
            f"[{self.instance_id}] 淘宝客户端已创建: device_id={device_id_desc}"
        )

        # 4. 并行运行：后端连接 + 淘宝监听
        logger.info(f"[{self.instance_id}] 开始运行: 连接后端 + 监听淘宝消息")
        await asyncio.gather(
            self.bridge.connect_forever(),
            self.live.main(),
        )

    async def _handle_taobao_message(  # pylint: disable=too-many-arguments,too-many-locals
        self,
        _websocket,
        cid,
        send_user_id,
        send_user_name,
        send_message,
        raw_message=None,
        message_chain=None,
    ):
        """淘宝收到消息 -> 通过桥接上报后端。

        现在上报包含完整的原始消息(raw)和消息链(message_chain)，
        确保后端保存时不会丢失任何信息。同时尽力从原始消息中提取
        商品信息(标题/价格)并作为结构化字段透传，避免商品字段丢失。
        """
        if not self.bridge:
            return

        # 从原始消息中提取商品信息(标题/价格)并作为结构化字段透传，
        # 避免 "商品" 字段在后端/下游(如 LangBot)丢失。
        item_title = ""
        item_price = ""
        item_info = {}
        try:
            if raw_message and isinstance(raw_message, dict):
                msg_body = raw_message.get("1", {}).get("10", {})
                if isinstance(msg_body, dict):
                    item_info = msg_body.get("itemInfo") or {}
                    if isinstance(item_info, dict):
                        item_title = item_info.get("title", "") or ""
                        item_price = item_info.get("price", "") or ""
        except Exception:  # pylint: disable=broad-except
            item_title = ""
            item_price = ""
            item_info = {}

        payload = {
            "platform_id": "taobao",
            "instance_id": self.instance_id,
            "conversation_id": cid,
            "sender_id": send_user_id,
            "sender_name": send_user_name,
            "message_type": "text",
            "message_content": send_message,
            "idempotency_key": f"taobao-{send_user_id}-{cid}-{int(time.time()*1000)}",
        }
        # 商品结构化字段（不丢失商品信息）
        if item_title or item_price or item_info:
            payload["item"] = item_info
            payload["item_title"] = item_title
            payload["item_price"] = item_price
        # 保存完整的原始平台消息
        if raw_message is not None:
            payload["raw"] = raw_message
        # 保存消息链；若商品信息存在，追加商品卡片元素
        if message_chain is not None:
            chain = list(message_chain)
            if item_title:
                chain.append({"type": "item", "title": item_title, "price": item_price})
            payload["message_chain"] = chain
        await self.bridge.send_inbound(payload)
        logger.info(
            f"[{self.instance_id}] Reported inbound message to backend "
            f"conversation={cid} sender={send_user_name} "
            f"item_title={item_title or '无'} item_price={item_price or '无'}"
        )

    async def _handle_backend_command(self, data: dict):
        """后端下发指令 -> 调用淘宝发送。"""
        if not self.live:
            return
        command_type = data.get("command_type") or data.get("type")
        payload = data.get("payload", {})
        if not payload or not self.live.ws:
            logger.warning(f"[{self.instance_id}] No payload or not connected")
            return

        # 兼容两种字段命名：cid/conversation_id、toid/target_id
        cid = payload.get("cid") or payload.get("conversation_id") or ""
        toid = payload.get("toid") or payload.get("target_id") or ""
        if not cid or not toid:
            logger.warning(
                f"[{self.instance_id}] 指令缺少 cid/toid: command={command_type} "
                f"payload_keys={list(payload.keys())}"
            )
            return

        sender_nick = f"cntaobao{self.live.nk}"

        if command_type in ("send_text", "send"):
            # 优先取 message_chain 中的文本，其次取 text/message_content
            text = payload.get("text") or payload.get("message_content", "")
            chain = payload.get("message_chain") or []
            if not text and isinstance(chain, list):
                for elem in chain:
                    if not isinstance(elem, dict):
                        continue
                    if elem.get("type") == "text":
                        content = elem.get("content")
                        if isinstance(content, dict):
                            text = content.get("text", "")
                        elif isinstance(content, str):
                            text = content
                        if text:
                            break
            if not text:
                logger.warning(
                    f"[{self.instance_id}] 指令缺少文本内容: command={command_type}"
                )
                return
            await self.live.send_msg(
                self.live.ws, cid, toid, sender_nick,
                {"type": "text", "text": text},
            )
        elif command_type == "send_image":
            await self.live.send_msg(
                self.live.ws, cid, toid, sender_nick,
                {
                    "type": "image",
                    "file_id": payload.get("file_id", ""),
                    "size": payload.get("size", 0),
                    "image_url": payload.get("image_url", ""),
                    "width": payload.get("width", 0),
                    "height": payload.get("height", 0),
                },
            )
        else:
            logger.warning(f"[{self.instance_id}] Unknown command: {command_type}")


# ASCII art banner, same as the E-SP-Line2 backend startup banner.
BANNER = r"""  # pylint: disable=line-too-long
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
    """程序入口：解析参数、并发启动所有淘宝实例并处理退出。"""
    args = parse_args(argv)
    instance_ids = [i.strip() for i in args.instance_id.split(",") if i.strip()]
    if not instance_ids:
        logger.error("No instance IDs provided")
        return 1

    print(BANNER, flush=True)
    logger.info(f"Starting Taobao adapter for instances: {instance_ids}")
    instances = [TaobaoInstance(args.backend, i, args.token) for i in instance_ids]

    # 并发启动所有实例(多开)
    tasks = [asyncio.create_task(inst.start()) for inst in instances]

    try:
        await asyncio.gather(*tasks)
    except KeyboardInterrupt:
        logger.info("Shutting down...")
        await shutdown(instances)
    except Exception as e:  # pylint: disable=broad-except
        logger.error(f"Fatal error: {e}", exc_info=True)
        await shutdown(instances)
        return 1
    return 0


async def shutdown(instances: List["TaobaoInstance"]) -> None:
    """优雅关闭所有实例的 lwp WebSocket 与 ESPL 桥接连接。"""
    for inst in instances:
        try:
            if inst.live:
                await inst.live.close()
        except Exception as e:  # pylint: disable=broad-except
            logger.warning(f"Error closing lwp connection: {e}")
        try:
            if inst.bridge:
                await inst.bridge.close()
        except Exception as e:  # pylint: disable=broad-except
            logger.warning(f"Error closing bridge connection: {e}")


if __name__ == "__main__":
    sys.exit(asyncio.run(main(sys.argv[1:])))

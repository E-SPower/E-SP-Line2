"""
E-SP-Line2 XianYu (闲鱼) Adapter Bridge
========================================

中间层：将闲鱼(XianYuApis)平台的原始消息转换为 ESPL v3 协议，
并通过 WebSocket 与 E-SP-Line2 后端通信。

职责：
1. 启动时从后端 HTTP API 拉取当前实例的配置(cookie / device_id)
2. 建立与后端 `/ws/adapter?instance_id=xxx` 的 WebSocket 长连接
3. 把闲鱼收到的消息转换为 ESPL payload 格式上报后端
4. 接收后端下发的出站指令(send_text / send_image)，调用闲鱼 API 发送
5. 支持多开：每个实例一个独立进程/线程，独立 Cookie 配置

用法：
    python main.py --instance-id <INSTANCE_ID> --backend http://localhost:8080
"""

from __future__ import annotations

import asyncio
import json
import time
import uuid
from dataclasses import dataclass, field
from typing import Any, Dict, Optional, Callable

import requests
import websockets

# ---------------------------------------------------------------- logging ---
from loguru import logger

# 详细日志格式：时间 | 级别 | 消息
logger.remove()
logger.add(
    sink=lambda msg: print(msg, end=""),
    format="<green>{time:YYYY-MM-DD HH:mm:ss}</green> | <level>{level: <8}</level> | <level>{message}</level>",
    colorize=False,
)


# ------------------------------------------------------------- data types ----
@dataclass
class InstanceConfig:
    """从后端拉取的实例配置。"""
    instance_id: str
    adapter_id: str
    platform_id: str
    name: str
    config: Dict[str, Any] = field(default_factory=dict)


# ------------------------------------------------------------ config fetch ---
def fetch_instance_config(backend_url: str, instance_id: str, token: str = "") -> InstanceConfig:
    """从后端 API 拉取指定实例的配置。"""
    url = f"{backend_url.rstrip('/')}/api/v1/instances/{instance_id}"
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    logger.info(f"[config] GET {url}")
    resp = requests.get(url, headers=headers, timeout=15)
    if resp.status_code == 401:
        raise RuntimeError(
            f"无法从后端拉取实例配置：401 未授权。实例 {instance_id} 可能不存在，"
            "或需要为接入器配置有效的后端登录 token(--token)。"
        )
    resp.raise_for_status()
    data = resp.json()

    config_raw = data.get("config", "")
    config_dict: Dict[str, Any] = {}
    if config_raw:
        try:
            parsed = json.loads(config_raw)
            if isinstance(parsed, dict):
                config_dict = parsed
        except Exception:
            logger.warning("Instance config is not valid JSON, treating as empty")

    # 打印配置概要（不泄露 cookie 值）
    cfg_summary = {k: ("******" if "cookie" in k.lower() else v) for k, v in config_dict.items()}
    logger.info(
        f"[config] 拉取成功: instance={data.get('id', instance_id)} "
        f"adapter={data.get('adapter_id', '')} platform={data.get('platform_id', '')} "
        f"name={data.get('name', '')} config_keys={list(config_dict.keys())} "
        f"config={cfg_summary}"
    )

    return InstanceConfig(
        instance_id=data.get("id", instance_id),
        adapter_id=data.get("adapter_id", ""),
        platform_id=data.get("platform_id", "taobao"),
        name=data.get("name", instance_id),
        config=config_dict,
    )


# ---------------------------------------------------------------- v3 helpers --
def make_envelope(event_type: str, platform: str, adapter_id: str, payload: Any) -> Dict[str, Any]:
    """构造 ESPL v3 消息信封。"""
    return {
        "protocol_version": "v3",
        "event_id": uuid.uuid4().hex,
        "trace_id": f"trace-{uuid.uuid4().hex[:16]}",
        "timestamp": int(time.time() * 1000),
        "platform": platform,
        "adapter_id": adapter_id,
        "event_type": event_type,
        "payload": payload,
    }


def inbound_message_payload(
    *,
    conversation_id: str,
    sender_id: str,
    sender_name: str,
    message_type: str,
    message_content: str,
    idempotency_key: str = "",
    raw: Optional[Dict[str, Any]] = None,
    message_chain: Optional[List[Dict[str, Any]]] = None,
    extra: Optional[Dict[str, Any]] = None,
) -> Dict[str, Any]:
    """构造后端 `persistInboundMessage` 期望的 payload 字段。

    包含完整的原始消息（raw）和消息链（message_chain），
    确保桥上报的所有信息都不会丢失。
    """
    payload: Dict[str, Any] = {
        "platform_id": "xianyu",
        "conversation_id": conversation_id,
        "sender_id": sender_id,
        "sender_name": sender_name,
        "message_type": message_type,
        "message_content": message_content,
        "idempotency_key": idempotency_key,
    }
    if raw is not None:
        payload["raw"] = raw
    if message_chain is not None:
        payload["message_chain"] = message_chain
    if extra:
        payload.update(extra)
    return payload


# -------------------------------------------------------------- bridge core ----
class EspBridge:
    """ESPL 中间层：维护与后端 WebSocket 的连接，负责消息收发。"""

    def __init__(
        self,
        backend_url: str,
        instance_id: str,
        token: str = "",
        reconnect_delay: int = 5,
        on_inbound: Optional[Callable[[Dict[str, Any]], None]] = None,
    ):
        self.backend_url = backend_url.rstrip("/")
        self.instance_id = instance_id
        self.token = token
        self.reconnect_delay = reconnect_delay
        self.on_inbound = on_inbound  # 收到后端下发指令时的回调
        self.ws: Optional[websockets.WebSocketClientProtocol] = None
        self.running = False

    def _ws_url(self) -> str:
        base = self.backend_url.replace("http://", "ws://").replace("https://", "wss://")
        return f"{base}/ws/adapter?instance_id={self.instance_id}"

    async def connect_forever(self):
        """持续连接后端 WebSocket，断线自动重连。"""
        self.running = True
        attempts = 0
        while self.running:
            try:
                attempts += 1
                logger.info(f"[ws] 连接后端 (第{attempts}次尝试): {self._ws_url()}")
                async with websockets.connect(self._ws_url()) as ws:
                    self.ws = ws
                    attempts = 0
                    logger.info("[ws] 后端 WebSocket 已连接")
                    await self._read_loop(ws)
            except asyncio.CancelledError:
                logger.info("[ws] 连接循环被取消")
                break
            except Exception as e:
                logger.error(f"[ws] 连接错误: {e}")
            if self.running:
                logger.info(f"[ws] {self.reconnect_delay}s 后重连...")
                await asyncio.sleep(self.reconnect_delay)
        logger.info("[ws] 连接循环结束")

    async def _read_loop(self, ws: websockets.WebSocketClientProtocol):
        """读取后端下发的消息(指令/ack)。"""
        async for raw in ws:
            try:
                data = json.loads(raw)
            except Exception:
                logger.warning(f"[ws] 收到非 JSON 消息: {raw[:200]}")
                continue

            msg_type = data.get("type", "")
            if msg_type == "connected":
                logger.info("[ws] 后端握手成功，就绪")
                continue
            if msg_type == "ack":
                logger.debug(f"[ws] 收到 ACK: event_id={data.get('event_id', '')}")
                continue

            # 出站指令
            logger.info(f"[ws] 收到后端指令: {data}")
            if self.on_inbound:
                try:
                    await self.on_inbound(data)
                except Exception as e:
                    logger.error(f"[ws] 指令处理出错: {e}")

    async def send_inbound(self, payload: Dict[str, Any]):
        """向后端上报一条入站消息。"""
        if not self.ws:
            logger.warning("[ws] 后端未连接，丢弃消息")
            return
        msg = json.dumps(payload, ensure_ascii=False)
        await self.ws.send(msg)
        logger.info(
            f"[ws] 已上报消息: conversation={payload.get('conversation_id', '')} "
            f"sender={payload.get('sender_name', '')} type={payload.get('message_type', '')} "
            f"len={len(msg)}"
        )

    async def close(self):
        self.running = False
        if self.ws:
            await self.ws.close()


# ------------------------------------------------------------------- entry ---
async def run_instance(
    backend_url: str,
    instance_id: str,
    token: str = "",
    on_command: Optional[Callable[[Dict[str, Any]], Any]] = None,
):
    """
    启动单个实例的桥接器。

    流程：
    1. 拉取实例配置
    2. 建立桥接器(连接后端 WS)
    3. 传入 on_command 回调：当后端下发 send_text/send_image 时调用
    """
    cfg = fetch_instance_config(backend_url, instance_id, token)
    logger.info(
        f"Instance loaded: {cfg.name} (adapter={cfg.adapter_id}, "
        f"platform={cfg.platform_id}, has_cookie={'cookie' in cfg.config})"
    )

    bridge = EspBridge(
        backend_url=backend_url,
        instance_id=instance_id,
        token=token,
        reconnect_delay=cfg.config.get("reconnect_delay", 5),
        on_inbound=on_command,
    )
    await bridge.connect_forever()
    return bridge

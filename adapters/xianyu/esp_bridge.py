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

    resp = requests.get(url, headers=headers, timeout=15)
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

    return InstanceConfig(
        instance_id=data.get("id", instance_id),
        adapter_id=data.get("adapter_id", ""),
        platform_id=data.get("platform_id", "xianyu"),
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
    extra: Optional[Dict[str, Any]] = None,
) -> Dict[str, Any]:
    """构造后端 `persistInboundMessage` 期望的 payload 字段。"""
    payload: Dict[str, Any] = {
        "platform_id": "xianyu",
        "conversation_id": conversation_id,
        "sender_id": sender_id,
        "sender_name": sender_name,
        "message_type": message_type,
        "message_content": message_content,
        "idempotency_key": idempotency_key,
    }
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
        while self.running:
            try:
                logger.info(f"Connecting to backend: {self._ws_url()}")
                async with websockets.connect(self._ws_url()) as ws:
                    self.ws = ws
                    logger.info("Backend WebSocket connected")
                    await self._read_loop(ws)
            except Exception as e:
                logger.error(f"WebSocket connection error: {e}")
            if self.running:
                logger.info(f"Reconnecting in {self.reconnect_delay}s...")
                await asyncio.sleep(self.reconnect_delay)

    async def _read_loop(self, ws: websockets.WebSocketClientProtocol):
        """读取后端下发的消息(指令/ack)。"""
        async for raw in ws:
            try:
                data = json.loads(raw)
            except Exception:
                logger.warning(f"Non-JSON message from backend: {raw}")
                continue

            msg_type = data.get("type", "")
            if msg_type == "connected":
                logger.info("Backend ready")
                continue
            if msg_type == "ack":
                continue

            # 出站指令
            if self.on_inbound:
                try:
                    await self.on_inbound(data)
                except Exception as e:
                    logger.error(f"Command handling error: {e}")

    async def send_inbound(self, payload: Dict[str, Any]):
        """向后端上报一条入站消息。"""
        if not self.ws:
            logger.warning("Backend not connected, dropping message")
            return
        await self.ws.send(json.dumps(payload, ensure_ascii=False))

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

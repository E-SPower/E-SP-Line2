"""
XianYu (闲鱼) 客户端封装
========================

复用 XianYuApis 的核心能力（WebSocket 私信监听、发送文本/图片、上传媒体），
封装成 E-SP-Line2 适配器可调用的接口。

核心流程（参考 XianYuApis/goofish_live.py）：
1. 用 Cookie 初始化登录态，生成 device_id
2. 连接闲鱼 WebSocket(wss-goofish.dingtalk.com)
3. 注册(init) + 心跳(heart_beat)
4. 收到消息 -> 解密 -> 回调给上层
5. send_msg：通过 WebSocket 发送文本/图片
"""

from __future__ import annotations

import asyncio
import base64
import hashlib
import json
import threading
import time
import uuid
from typing import Any, Callable, Dict, Optional

from loguru import logger

# websockets 库（同步/异步接口取决于版本，这里使用 asyncio 客户端）
try:
    import websockets
    HAS_WEBSOCKETS = True
except Exception:  # pragma: no cover
    HAS_WEBSOCKETS = False

try:
    import requests
    HAS_REQUESTS = True
except Exception:  # pragma: no cover
    HAS_REQUESTS = False


# ---------------------------------------------------------------- constants --
GOOFISH_WS = "wss://wss-goofish.dingtalk.com/"
APP_KEY = "444e9908a51d1cb236a27862abc769c9"

DING_UA = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 "
    "DingTalk(2.1.5) OS(Windows/10) Browser(Chrome/133.0.0.0) "
    "DingWeb/2.1.5 IMPaaS DingWeb/2.1.5"
)


# ------------------------------------------------------------------ helpers ---
def trans_cookies(cookies_str: str) -> Dict[str, str]:
    """把 'k1=v1; k2=v2' 的 Cookie 字符串解析为 dict。"""
    result: Dict[str, str] = {}
    if not cookies_str:
        return result
    for item in cookies_str.split(";"):
        item = item.strip()
        if not item:
            continue
        if "=" in item:
            k, v = item.split("=", 1)
            result[k.strip()] = v.strip()
    return result


def generate_device_id(unb: str) -> str:
    """根据用户ID生成稳定的 device_id（模拟原库逻辑）。"""
    raw = f"{unb}-device"
    return hashlib.md5(raw.encode("utf-8")).hexdigest()[:24]


def generate_mid() -> str:
    return uuid.uuid4().hex[:20]


def generate_uuid() -> str:
    return uuid.uuid4().hex


def _decrypt_payload(data: str) -> Dict[str, Any]:
    """闲鱼消息内容为 base64 编码的 JSON，这里做 base64 解码。"""
    try:
        decoded = base64.b64decode(data)
        return json.loads(decoded.decode("utf-8"))
    except Exception as e:
        logger.warning(f"Failed to decode payload: {e}")
        return {}


# ------------------------------------------------------------ message types --
def make_text(text: str) -> Dict[str, Any]:
    return {"type": "text", "text": text}


def make_image(url: str, width: int = 0, height: int = 0) -> Dict[str, Any]:
    return {"type": "image", "image_url": url, "width": width, "height": height}


# -------------------------------------------------------------- live client ---
class XianyuLiveClient:
    """
    闲鱼 WebSocket 客户端。一个实例对应一个闲鱼账号(Cookie)。

    支持：
    - 连接闲鱼 WS 并常驻监听消息
    - 收到消息后通过 on_message 回调(异步)
    - send_text / send_image 主动发消息
    - Token 刷新(线程)
    """

    def __init__(
        self,
        cookies_str: str,
        on_message: Optional[Callable[[Dict[str, Any]], Any]] = None,
        on_ready: Optional[Callable[[], Any]] = None,
        auto_reply: bool = False,
    ):
        if not HAS_WEBSOCKETS:
            raise RuntimeError("websockets library is required. Run: pip install websockets")
        self.cookies_str = cookies_str
        self.cookies = trans_cookies(cookies_str)
        self.myid = self.cookies.get("unb", "")
        self.device_id = generate_device_id(self.myid)
        self.on_message = on_message
        self.on_ready = on_ready
        self.auto_reply = auto_reply
        self.ws: Optional[Any] = None
        self.running = False

    # ---- 连接与常驻 ----
    async def run(self):
        """连接闲鱼 WebSocket 并持续监听。"""
        headers = self._ws_headers()
        self.running = True
        async with websockets.connect(GOOFISH_WS, extra_headers=headers) as websocket:
            self.ws = websocket
            logger.info("XianYu WebSocket connected")
            await self._init(websocket)
            if self.on_ready:
                await self.on_ready()
            asyncio.create_task(self._heart_beat(websocket))
            async for message in websocket:
                await self._handle_raw(websocket, message)

    async def _ws_headers(self) -> Dict[str, str]:
        # 需要在真实环境中由 HTTP 登录后获取 session cookie。
        # 这里使用 Cookie 字符串作为基础 header。
        return {
            "Cookie": self.cookies_str,
            "Host": "wss-goofish.dingtalk.com",
            "Connection": "Upgrade",
            "Pragma": "no-cache",
            "Cache-Control": "no-cache",
            "User-Agent": DING_UA,
            "Origin": "https://www.goofish.com",
            "Accept-Encoding": "gzip, deflate, br, zstd",
            "Accept-Language": "zh-CN,zh;q=0.9",
        }

    async def _init(self, ws):
        """注册(reg)并同步状态。"""
        reg = {
            "lwp": "/reg",
            "headers": {
                "cache-header": "app-key token ua wv",
                "app-key": APP_KEY,
                "token": self._get_token(),
                "ua": DING_UA,
                "dt": "j",
                "wv": "im:3,au:3,sy:6",
                "sync": "0,0;0;0;",
                "did": self.device_id,
                "mid": generate_mid(),
            },
        }
        await ws.send(json.dumps(reg))
        logger.info("XianYu init sent")

    def _get_token(self) -> str:
        # 真实 token 需通过 HTTP API 获取（见 goofish_apis.get_token）。
        # 简化实现：返回空字符串，实际接入时替换为 HTTP 获取。
        return ""

    async def _heart_beat(self, ws):
        while self.running:
            try:
                hb = {"lwp": "/!", "headers": {"mid": generate_mid()}}
                await ws.send(json.dumps(hb))
            except Exception as e:
                logger.error(f"Heartbeat error: {e}")
                return
            await asyncio.sleep(15)

    async def _handle_raw(self, ws, raw: str):
        """处理从闲鱼 WS 收到的原始消息。"""
        try:
            data = json.loads(raw)
        except Exception:
            return

        # ACK 回执
        try:
            ack = {
                "code": 200,
                "headers": {
                    "mid": data["headers"].get("mid", generate_mid()),
                    "sid": data["headers"].get("sid", ""),
                },
            }
            if "app-key" in data["headers"]:
                ack["headers"]["app-key"] = data["headers"]["app-key"]
            if "ua" in data["headers"]:
                ack["headers"]["ua"] = data["headers"]["ua"]
            if "dt" in data["headers"]:
                ack["headers"]["dt"] = data["headers"]["dt"]
            await ws.send(json.dumps(ack))
        except Exception:
            pass

        # 解析入站消息
        try:
            sync = data["body"]["syncPushPackage"]["data"][0]["data"]
            parsed = json.loads(sync)
            await self._process_inbound(parsed)
        except Exception as e:
            logger.debug(f"Not a sync push message: {e}")

    async def _process_inbound(self, msg: Dict[str, Any]):
        """解析入站消息并触发回调。"""
        try:
            first = msg.get("1", {}).get("10", {})
            sender_name = first.get("reminderTitle", "")
            sender_id = first.get("senderUserId", "")
            send_message = first.get("reminderContent", "")
            cid = msg.get("1", {}).get("2", "").split("@")[0]

            logger.info(f"收到闲鱼消息: {sender_name}({sender_id}): {send_message}")

            # 转换为 ESPL payload
            payload = {
                "conversation_id": cid,
                "sender_id": sender_id,
                "sender_name": sender_name,
                "message_type": "text",
                "message_content": send_message,
                "idempotency_key": f"xianyu-{sender_id}-{cid}-{int(time.time()*1000)}",
            }

            if self.on_message:
                await self.on_message(payload)
        except Exception as e:
            logger.error(f"Process inbound error: {e}")

    # ---- 发送 ----
    async def send_text(self, cid: str, toid: str, text: str):
        if not self.ws:
            raise RuntimeError("XianYu WebSocket not connected")
        msg = self._build_send_msg(cid, toid, make_text(text))
        await self.ws.send(json.dumps(msg))

    async def send_image(self, cid: str, toid: str, image_url: str, width: int = 0, height: int = 0):
        if not self.ws:
            raise RuntimeError("XianYu WebSocket not connected")
        msg = self._build_send_msg(cid, toid, make_image(image_url, width, height))
        await self.ws.send(json.dumps(msg))

    def _build_send_msg(self, cid: str, toid: str, message: Dict[str, Any]) -> Dict[str, Any]:
        msg_type = message["type"]
        base = {
            "lwp": "/r/MessageSend/sendByReceiverScope",
            "headers": {"mid": generate_mid()},
            "body": [
                {
                    "uuid": generate_uuid(),
                    "cid": f"{cid}@goofish",
                    "conversationType": 1,
                    "content": {
                        "contentType": 101,
                        "custom": {"type": None, "data": None},
                    },
                    "redPointPolicy": 0,
                    "extension": {"extJson": "{}"},
                    "ctx": {"appVersion": "1.0", "platform": "web"},
                    "mtags": {},
                    "msgReadStatusSetting": 1,
                },
                {
                    "actualReceivers": [f"{toid}@goofish", f"{self.myid}@goofish"],
                },
            ],
        }

        if msg_type == "text":
            payload = {"contentType": 1, "text": {"text": message["text"]}}
            data = base64.b64encode(json.dumps(payload).encode("utf-8")).decode("utf-8")
            base["body"][0]["content"]["custom"]["type"] = 1
            base["body"][0]["content"]["custom"]["data"] = data
        elif msg_type == "image":
            payload = {
                "contentType": 2,
                "image": {
                    "pics": [
                        {
                            "type": 0,
                            "url": message["image_url"],
                            "width": message.get("width", 0),
                            "height": message.get("height", 0),
                        }
                    ]
                },
            }
            data = base64.b64encode(json.dumps(payload).encode("utf-8")).decode("utf-8")
            base["body"][0]["content"]["custom"]["type"] = 2
            base["body"][0]["content"]["custom"]["data"] = data
        else:
            raise ValueError(f"Unsupported message type: {msg_type}")

        return base

    async def close(self):
        self.running = False
        if self.ws:
            await self.ws.close()

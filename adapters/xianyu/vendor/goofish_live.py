import base64
import json
import asyncio
import threading
import time

from loguru import logger
import websockets
from goofish_apis import XianyuApis

# 确保日志立即刷新到文件（E-SP-Line2 通过重定向 stdout 收集日志，
# Python 默认块缓冲会导致日志延迟/丢失，这里强制 flush）。
logger.remove()
logger.add(
    sink=lambda msg: print(msg, end="", flush=True),
    format="<green>{time:YYYY-MM-DD HH:mm:ss}</green> | <level>{level: <8}</level> | <level>{message}</level>",
    colorize=False,
)

from utils.goofish_utils import generate_mid, generate_uuid, trans_cookies, generate_device_id, decrypt, \
    get_session_cookies_str
from message import Message, make_text, make_image


# websockets 14+ 将 extra_headers 改名为 additional_headers；这里做兼容封装。
# 返回可异步上下文管理的连接对象（websockets.connect 本身就是 async CM）。
def _ws_connect(uri, headers):
    try:
        # 新版 websockets (>=14)
        return websockets.connect(uri, additional_headers=headers)
    except TypeError:
        # 旧版 websockets (<14) 使用 extra_headers
        return websockets.connect(uri, extra_headers=headers)


class XianyuLive:
    def __init__(self, cookies_str, message_callback=None, device_id_override=None):
        self.base_url = 'wss://wss-goofish.dingtalk.com/'
        self.cookies_str = cookies_str
        self.cookies = trans_cookies(cookies_str)
        self.myid = self.cookies.get('unb', '')
        # 支持外部注入 device_id（多开多配置时每个实例可独立指定）
        self.device_id = device_id_override or generate_device_id(self.myid)
        self.xianyu = XianyuApis(self.cookies, self.device_id)
        self.ws = None
        # 消息回调钩子：async (message, websocket, cid, send_user_id, send_user_name, send_message) -> None
        self.message_callback = message_callback
        self._instance_id = ''
        self._bridge = None

    def bind_bridge(self, instance_id, bridge):
        """绑定 ESPL 桥接器，用于上报消息与接收指令。"""
        self._instance_id = instance_id
        self._bridge = bridge

    async def close(self):
        """关闭 WebSocket 连接(供优雅退出)。"""
        if self.ws:
            try:
                await self.ws.close()
            except Exception:
                pass
            self.ws = None

    async def list_all_conversations(self, cid):
        headers = {
            "Cookie": get_session_cookies_str(self.xianyu.session),
            "Host": "wss-goofish.dingtalk.com",
            "Connection": "Upgrade",
            "Pragma": "no-cache",
            "Cache-Control": "no-cache",
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
            "Origin": "https://www.goofish.com",
            "Accept-Encoding": "gzip, deflate, br, zstd",
            "Accept-Language": "zh-CN,zh;q=0.9",
        }
        async with _ws_connect(self.base_url, headers) as websocket:
            asyncio.create_task(self.init(websocket))
            send_mid = generate_mid()
            msg = {
                "lwp": "/r/MessageManager/listUserMessages",
                "headers": {
                    "mid": send_mid
                },
                "body": [
                    f"{cid}@goofish",
                    False,
                    9007199254740991,
                    20,
                    False
                ]
            }
            user_message_models = []
            async for message in websocket:
                try:
                    message = json.loads(message)
                    ack = {
                        "code": 200,
                        "headers": {
                            "mid": message["headers"]["mid"] if "mid" in message["headers"] else generate_mid(),
                            "sid": message["headers"]["sid"] if "sid" in message["headers"] else '',
                        }
                    }
                    if 'app-key' in message["headers"]:
                        ack["headers"]["app-key"] = message["headers"]["app-key"]
                    if 'ua' in message["headers"]:
                        ack["headers"]["ua"] = message["headers"]["ua"]
                    if 'dt' in message["headers"]:
                        ack["headers"]["dt"] = message["headers"]["dt"]
                    await websocket.send(json.dumps(ack))
                except Exception as e:
                    pass
                try:
                    if 'lwp' in message and message['lwp'] == "/s/vulcan":
                        await websocket.send(json.dumps(msg))
                    recv_mid = message["headers"]["mid"] if "mid" in message["headers"] else ''
                    if recv_mid == send_mid:
                        logger.info(f"user history message: {message}")
                        has_more = message["body"]["hasMore"] == 1
                        next_cursor = message["body"]["nextCursor"]
                        for user_message in message["body"]["userMessageModels"]:
                            send_user_name = user_message["message"]["extension"]["reminderTitle"]
                            send_user_id = user_message["message"]["extension"]["senderUserId"]
                            send_message_base64 = user_message["message"]["content"]["custom"]["data"]
                            send_message_json = json.loads(base64.b64decode(send_message_base64).decode('utf-8'))
                            user_message_models.insert(0, {
                                "send_user_id": send_user_id,
                                "send_user_name": send_user_name,
                                "message": send_message_json
                            })
                        if has_more:
                            logger.info(f"has more history messages, next cursor: {next_cursor}")
                            send_mid = generate_mid()
                            msg["headers"]["mid"] = send_mid
                            msg["body"][2] = next_cursor
                            await websocket.send(json.dumps(msg))
                        else:
                            return user_message_models
                except Exception as e:
                    return user_message_models

    async def create_chat(self, ws, toid, item_id='891198795482'):
        msg = {
            "lwp": "/r/SingleChatConversation/create",
            "headers": {
                "mid": generate_mid()
            },
            "body": [
                {
                    "pairFirst": f"{toid}@goofish",
                    "pairSecond": f"{self.myid}@goofish",
                    "bizType": "1",
                    "extension": {
                        "itemId": item_id
                    },
                    "ctx": {
                        "appVersion": "1.0",
                        "platform": "web"
                    }
                }
            ]
        }
        await ws.send(json.dumps(msg))

    async def send_msg(self, ws, cid, toid, message: Message):
        msg_type = message["type"]
        msg = {
            "lwp": "/r/MessageSend/sendByReceiverScope",
            "headers": {
                "mid": generate_mid()
            },
            "body": [
                {
                    "uuid": generate_uuid(),
                    "cid": f"{cid}@goofish",
                    "conversationType": 1,
                    "content": {
                        "contentType": 101,
                        "custom": {
                            "type": None,
                            "data": None
                        }
                    },
                    "redPointPolicy": 0,
                    "extension": {
                        "extJson": "{}"
                    },
                    "ctx": {
                        "appVersion": "1.0",
                        "platform": "web"
                    },
                    "mtags": {},
                    "msgReadStatusSetting": 1
                },
                {
                    "actualReceivers": [
                        f"{toid}@goofish",
                        f"{self.myid}@goofish"
                    ]
                }
            ]
        }
        if msg_type == "text":
            payload = {
                "contentType": 1,
                "text": {
                    "text": message["text"]
                }
            }
            text_base64 = str(base64.b64encode(json.dumps(payload).encode('utf-8')), 'utf-8')
            msg["body"][0]["content"]["custom"]["type"] = 1
            msg["body"][0]["content"]["custom"]["data"] = text_base64
        elif msg_type == "image":
            payload = {
                "contentType": 2,
                "image": {
                    "pics": [
                        {
                            "type": 0,
                            "url": message["image_url"],
                            "width": message["width"],
                            "height": message["height"]
                        }
                    ]
                }
            }
            image_base64 = str(base64.b64encode(json.dumps(payload).encode('utf-8')), 'utf-8')
            msg["body"][0]["content"]["custom"]["type"] = 2
            msg["body"][0]["content"]["custom"]["data"] = image_base64
        elif msg_type == "audio":
            # TODO: handle audio message
            logger.error(f"不支持的消息类型: {msg_type}")
            return
        else:
            logger.error(f"不支持的消息类型: {msg_type}")
            return
        await ws.send(json.dumps(msg))

    async def init(self, ws):
        try:
            data = self.xianyu.get_token()
        except Exception as e:
            logger.error(f"闲鱼登录失败: {e}")
            logger.error("实例无法启动，请检查 Cookie 是否有效。")
            raise RuntimeError(f"闲鱼登录失败: {e}")
        token = data['data']['accessToken'] if 'data' in data and 'accessToken' in data['data'] else ''
        if not token:
            logger.error('获取token失败：Cookie 无效或已过期，请重新填写闲鱼 Cookie')
            raise RuntimeError("获取token失败：Cookie 无效或已过期")
        # 登录成功日志：显示用户信息与时间
        import datetime as _dt
        tracknick = self.cookies.get('tracknick', '')
        try:
            import urllib.parse
            tracknick = urllib.parse.unquote(tracknick)
        except Exception:
            pass
        logger.info(
            f"[闲鱼] 登录成功: 用户={tracknick or '未知'} unb={self.myid} "
            f"时间={_dt.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
        )
        msg = {
            "lwp": "/reg",
            "headers": {
                "cache-header": "app-key token ua wv",
                "app-key": "444e9908a51d1cb236a27862abc769c9",
                "token": token,
                "ua": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 DingTalk(2.1.5) OS(Windows/10) Browser(Chrome/133.0.0.0) DingWeb/2.1.5 IMPaaS DingWeb/2.1.5",
                "dt": "j",
                "wv": "im:3,au:3,sy:6",
                "sync": "0,0;0;0;",
                "did": self.device_id,
                "mid": generate_mid()
            }
        }
        await ws.send(json.dumps(msg))
        current_time = int(time.time() * 1000)
        msg = {
            "lwp": "/r/SyncStatus/ackDiff",
            "headers": {"mid": generate_mid()},
            "body": [
                {
                    "pipeline": "sync",
                    "tooLong2Tag": "PNM,1",
                    "channel": "sync",
                    "topic": "sync",
                    "highPts": 0,
                    "pts": current_time * 1000,
                    "seq": 0,
                    "timestamp": current_time
                }
            ]
        }
        await ws.send(json.dumps(msg))
        logger.info('init')

    async def heart_beat(self, ws):
        while True:
            msg = {
                "lwp": "/!",
                "headers": {
                    "mid": generate_mid()
                 }
            }
            await ws.send(json.dumps(msg))
            await asyncio.sleep(15)

    def user_alive(self):
        while True:
            try:
                time.sleep(600)
                self.xianyu.refresh_token()
            except Exception as e:
                logger.error(f"保活线程异常: {e}")
                time.sleep(60)

    async def main(self):
        headers = {
            "Cookie": get_session_cookies_str(self.xianyu.session),
            "Host": "wss-goofish.dingtalk.com",
            "Connection": "Upgrade",
            "Pragma": "no-cache",
            "Cache-Control": "no-cache",
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
            "Origin": "https://www.goofish.com",
            "Accept-Encoding": "gzip, deflate, br, zstd",
            "Accept-Language": "zh-CN,zh;q=0.9",
        }
        threading.Thread(target=self.user_alive).start()
        import datetime as _dt
        logger.info(
            f"[闲鱼] 正在连接 WebSocket: {self.base_url} "
            f"时间={_dt.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
        )
        try:
        	async with _ws_connect(self.base_url, headers) as websocket:
        		# 保存 WebSocket 引用，供 _handle_backend_command 发送消息时使用
        		self.ws = websocket
        		logger.info(
        			f"[闲鱼] WebSocket 已连接: {self.base_url} "
        			f"时间={_dt.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
        		)
        		asyncio.create_task(self.init(websocket))
        		asyncio.create_task(self.heart_beat(websocket))
        		async for message in websocket:
        			# logger.info(f"message: {message}")
        			message = json.loads(message)
        			ack = {
        				"code": 200,
        				"headers": {
        					"mid": message["headers"]["mid"] if "mid" in message["headers"] else generate_mid(),
        					"sid": message["headers"]["sid"] if "sid" in message["headers"] else '',
        				}
        			}
        			if 'app-key' in message["headers"]:
        				ack["headers"]["app-key"] = message["headers"]["app-key"]
        			if 'ua' in message["headers"]:
        				ack["headers"]["ua"] = message["headers"]["ua"]
        			if 'dt' in message["headers"]:
        				ack["headers"]["dt"] = message["headers"]["dt"]
        			await websocket.send(json.dumps(ack))
      
        			await self.handle_message(message, websocket)
        except Exception as e:
        	# 断开连接时清除 ws 引用
        	self.ws = None
        	logger.error(
        		f"[闲鱼] WebSocket 连接断开: {e} "
        		f"时间={_dt.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
        	)
        	raise

    async def handle_message(self, message, websocket):
        import datetime as _dt
        try:
            data = message["body"]["syncPushPackage"]["data"][0]["data"]
            data = json.loads(data)
            # logger.info(f"无需解密 message: {data}")
        except Exception as e:
            try:
                raw_decrypted = decrypt(data)
                raw_message = json.loads(raw_decrypted)
                # logger.info(f"解密的 message: {raw_message}")

                send_user_name = raw_message["1"]["10"]["reminderTitle"]
                send_user_id = raw_message["1"]["10"]["senderUserId"]
                send_message = raw_message["1"]["10"]["reminderContent"]

                # 过滤自己的消息（不处理自己发送的消息）
                if send_user_id == self.myid:
                    logger.debug(
                        f"[闲鱼] 忽略自己的消息: sender={send_user_name}({send_user_id}) "
                        f"content={send_message[:20]}"
                    )
                    return

                # 从 reminderUrl 提取关联的商品 ID
                item_id = ''
                item_title = ''
                item_price = ''
                try:
                    reminder_url = raw_message["1"]["10"].get("reminderUrl", "")
                    if 'itemId=' in reminder_url:
                        item_id = reminder_url.split('itemId=')[1].split('&')[0]
                except Exception:
                    pass

                logger.info(
                    f"[闲鱼] 收到新消息: 发送者={send_user_name}({send_user_id}) "
                    f"内容={send_message} 商品ID={item_id or '无'} "
                    f"时间={_dt.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
                )

                cid = raw_message["1"]["2"]
                cid = cid.split('@')[0]
                logger.info(f"[闲鱼] 会话 cid={cid} 发送者={send_user_id}")

                # 构建消息链：如果有关联商品，追加商品信息元素
                message_chain = [{"type": "text", "text": send_message}]
                if item_id:
                    message_chain.append({
                        "type": "item",
                        "title": item_title or '',
                        "price": item_price or '',
                        "item_id": item_id,
                    })

                # 触发外部回调（ESPL 桥接上报 / 业务处理）
                if self.message_callback:
                    try:
                        await self.message_callback(
                            websocket, cid, send_user_id,
                            send_user_name, send_message,
                            raw_message=raw_message,
                            message_chain=message_chain,
                        )
                    except Exception as cb_e:
                        logger.error(f"message_callback error: {cb_e}")

                # 若绑定了桥接器，由桥接器统一上报；否则维持本地 echo 便于测试
                elif self._bridge:
                    payload = {
                        "platform_id": "xianyu",
                        "conversation_id": cid,
                        "sender_id": send_user_id,
                        "sender_name": send_user_name,
                        "message_type": "text",
                        "message_content": send_message,
                        "raw": raw_message,  # 保存完整的原始平台消息
                        "message_chain": message_chain,  # 保存消息链
                        "idempotency_key": f"xianyu-{send_user_id}-{cid}-{int(time.time()*1000)}",
                    }
                    try:
                        await self._bridge.send_inbound(payload)
                    except Exception as bridge_e:
                        logger.error(f"bridge send error: {bridge_e}")

                # 兜底：本地 echo 回复（纯测试模式）
                else:
                    reply = f'{send_user_name} 说了: {send_message}'
                    await self.send_msg(websocket, cid, send_user_id, make_text(reply))

                # 回复图片
                # res_json = self.xianyu.upload_media(r"D:\Desktop\1.png")
                # image_object = res_json["object"]
                # width, height = map(int, image_object["pix"].split('x'))
                # await self.send_msg(websocket, cid, send_user_id, make_image(image_object["url"], width, height))
            except Exception as e:
                pass

if __name__ == '__main__':
    # 1 获取全部聊天记录
    # cid = '47812870000'
    # all_messages = asyncio.run(xianyuLive.list_all_conversations(cid))
    # for message in all_messages:
    #     print(message)

    # 2 常驻进程 用于接收消息和自动回复
    xianyuLive = XianyuLive(cookies_str='')
    asyncio.run(xianyuLive.main())

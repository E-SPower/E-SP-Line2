from typing import Annotated, Literal, Union, Optional
from typing_extensions import TypedDict
from pydantic import Field, TypeAdapter

class TextContent(TypedDict):
    type: Literal["text"]
    text: str

class ImageContent(TypedDict):
    type: Literal["image"]
    image_url: str
    width: int
    height: int

class AudioContent(TypedDict):
    type: Literal["audio"]
    audio_url: str
    duration_ms: int

Message = Annotated[
    Union[TextContent, ImageContent, AudioContent],
    Field(discriminator="type"),
]

message_adapter = TypeAdapter(Message)
def make_text(text: str) -> TextContent:
    return {"type": "text", "text": text}

def make_image(url: str, width: int = 0, height: int = 0) -> ImageContent:
    return {"type": "image", "image_url": url, "width": width, "height": height}

def make_audio(url: str, duration_ms: int = 0) -> AudioContent:
    return {"type": "audio", "audio_url": url, "duration_ms": duration_ms}

class Price(TypedDict, total=False):
    current_price: float
    original_price: float

class DeliverySettings(TypedDict, total=False):
    # 包邮\按距离计费\一口价\无需邮寄 四选一
    choice: Literal["包邮", "按距离计费", "一口价", "无需邮寄"]
    post_price: float
    can_self_pickup: bool


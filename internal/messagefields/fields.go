package messagefields

// 归一化对象的总类。catbus 用“资源”表达同一件事在所有平台上的同义名。
const (
	KindUser         = "user"
	KindItem         = "item"
	KindMedia        = "media"
	KindPrice        = "price"
	KindComment      = "comment"
	KindLive         = "live"
	KindEvent        = "event"
	KindGift         = "gift"
	KindRank         = "rank"
	KindConversation = "conversation"
	KindMessage      = "message"
	KindNotice       = "notice"
	KindAuthStatus   = "auth_status"
	KindAccount      = "account"
	KindFile         = "file"
)

// ItemKind 是内容对象的形态。
const (
	ItemKindVideo   = "video"
	ItemKindImage   = "image"
	ItemKindText    = "text"
	ItemKindArticle = "article"
	ItemKindGoods   = "goods"
	ItemKindTorrent = "torrent"
)

// ItemStatus 是内容对象的状态。只有自己的内容才带状态，取不到时为 null。
const (
	ItemStatusPublished = "published"
	ItemStatusReviewing = "reviewing"
	ItemStatusRejected  = "rejected"
	ItemStatusPrivate   = "private"
	ItemStatusDraft     = "draft"
	ItemStatusOnSale    = "on_sale"
	ItemStatusSold      = "sold"
	ItemStatusOffShelf  = "off_shelf"
)

// MessageType 是私信消息的内容形态。
const (
	MessageTypeText  = "text"
	MessageTypeImage = "image"
	MessageTypeVideo = "video"
	MessageTypeCard  = "card"
	MessageTypeOther = "other"
)

// NoticeType 是通知的类型。
const (
	NoticeTypeComment = "comment"
	NoticeTypeMention = "mention"
	NoticeTypeLike    = "like"
	NoticeTypeFollow  = "follow"
	NoticeTypeSystem  = "system"
)

// EventType 是直播事件。
const (
	EventTypeChat   = "chat"
	EventTypeGift   = "gift"
	EventTypeLike   = "like"
	EventTypeEnter  = "enter"
	EventTypeFollow = "follow"
	EventTypeOther  = "other"
)

// LiveStatus 是直播状态。
const (
	LiveStatusLive    = "live"
	LiveStatusOffline = "offline"
)

// 元素类型。入站消息的正文由若干元素组成，顺序有意义。
const (
	ElementText        = "text"
	ElementImage       = "image"
	ElementAudio       = "audio"
	ElementVideo       = "video"
	ElementFile        = "file"
	ElementProductCard = "product_card"
	ElementOrderInfo   = "order_info"
	ElementInquiry     = "inquiry"
	ElementLocation    = "location"
	ElementEmoji       = "emoji"
	ElementMention     = "mention"
)

// Field 是一个字段的契约。
type Field struct {
	// Path 是点分路径，数组下标用 []。
	Path string `json:"path"`
	// Type 是 JSON 类型：string / number / integer / boolean / object / array 或 `X|null` 这样的联合。
	Type string `json:"type"`
	// Nullable 表示取不到时是 null。
	Nullable bool `json:"nullable"`
	// Required 表示字段总是存在。
	Required bool `json:"required"`
	// Values 是枚举取值，空表示不限。
	Values []string `json:"values,omitempty"`
	// Summary 是字段含义与来源。
	Summary string `json:"summary"`
	// MessageField 标记这个字段属于消息正文（可被适配器填充）。
	MessageField bool `json:"message_field,omitempty"`
}

// Object 是一个归一化对象的字段契约。
type Object struct {
	Kind   string  `json:"kind"`
	Fields []Field `json:"fields"`
	// Notes 说明跨平台的恒定值、恒为 null 的字段等。
	Notes []string `json:"notes,omitempty"`
}

// Fields 返回到对象全部字段（含嵌套路径）的扁平视图。
func (o *Object) Field(path string) (Field, bool) {
	for _, f := range o.Fields {
		if f.Path == path {
			return f, true
		}
	}
	return Field{}, false
}

// MessageFields 返回消息正文可填充的字段。
func (o *Object) MessageFields() []Field {
	out := make([]Field, 0, len(o.Fields))
	for _, f := range o.Fields {
		if f.MessageField {
			out = append(out, f)
		}
	}
	return out
}

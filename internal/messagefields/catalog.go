package messagefields

// 字段契约。结构照抄 catbus src/core/schemas.ts 的归一化类型，说明部分补上字段来源与恒空条件。
//
// 约定：字段总是存在；取不到时按 Nullable 给 null（数组给 []）。计数是整数或 null，
// 时间是带时区的 ISO 8601。所有对象都带 id 和 url，可以直接作为下一条命令的参数。

var userRef = Object{
	Kind: "user_ref",
	Fields: []Field{
		{Path: "id", Type: "string", Required: true, Summary: "用户 ID。平台给的 ID 一律按字符串保存，64 位整数也不丢精度。", MessageField: true},
		{Path: "name", Type: "string", Nullable: true, Required: true, Summary: "昵称。接口不返回时为 null，例如 X 的私信会话只给 ID。", MessageField: true},
		{Path: "url", Type: "string", Nullable: true, Required: true, Summary: "主页链接。平台没有主页概念（例如 M-Team 站内信）时为 null。"},
	},
}

var media = Object{
	Kind: KindMedia,
	Fields: []Field{
		{Path: "id", Type: "string", Nullable: true, Required: true, Summary: "平台侧的媒体 ID。只有平台给了才填，例如 M-Team 没有。"},
		{Path: "type", Type: "string", Required: true, Values: []string{"image", "video", "audio"}, Summary: "媒体类型。"},
		{Path: "url", Type: "string", Nullable: true, Required: true, Summary: "可访问地址。刚上传、平台还在转码时为 null。", MessageField: true},
		{Path: "width", Type: "integer", Nullable: true, Required: true, Summary: "像素宽。平台不返回时为 null。"},
		{Path: "height", Type: "integer", Nullable: true, Required: true, Summary: "像素高。平台不返回时为 null。"},
		{Path: "duration", Type: "integer", Nullable: true, Required: true, Summary: "时长（秒）。图片和平台不返回时（例如直接给 URL 的卡片图）为 null。"},
	},
	Notes: []string{
		"媒体只描述内容本身；发送方的展示字段见 useref。",
		"同一个消息有多张图时按平台顺序排，顺序有意义。",
	},
}

var price = Object{
	Kind: KindPrice,
	Fields: []Field{
		{Path: "amount", Type: "number", Required: true, Summary: "金额，以元等主单位计，不是分。"},
		{Path: "currency", Type: "string", Required: true, Summary: "ISO 4217 币种，例如 CNY、JPY。平台只给「¥」时按平台默认币种补全。"},
	},
}

var user = Object{
	Kind: KindUser,
	Fields: append([]Field{
		{Path: "handle", Type: "string", Nullable: true, Required: true, Summary: "平台内的唯一短名（抖音的 unique_id、X 的 screen_name 等）。没有时为 null。"},
		{Path: "avatar", Type: "string", Nullable: true, Required: true, Summary: "头像地址。"},
		{Path: "bio", Type: "string", Nullable: true, Required: true, Summary: "简介。"},
		{Path: "stats", Type: "object", Required: true, Summary: "计数容器。"},
		{Path: "stats.followers", Type: "integer", Nullable: true, Required: true, Summary: "粉丝数。"},
		{Path: "stats.following", Type: "integer", Nullable: true, Required: true, Summary: "关注数。"},
		{Path: "stats.items", Type: "integer", Nullable: true, Required: true, Summary: "作品数。"},
		{Path: "stats.likes", Type: "integer", Nullable: true, Required: true, Summary: "获赞数。"},
	}, userRef.Fields...),
	Notes: []string{
		"user 在 user_ref 之外多出 handle、avatar、bio 和 stats。",
		"stats.* 常常恒为 null：B 站的上游用户接口不带计数，快手的统计字段上游没有。",
	},
}

var item = Object{
	Kind: KindItem,
	Fields: []Field{
		{Path: "id", Type: "string", Required: true, Summary: "内容 ID。BV 号、aweme_id、note_id 等原样保留。"},
		{Path: "kind", Type: "string", Required: true, Values: []string{ItemKindVideo, ItemKindImage, ItemKindText, ItemKindArticle, ItemKindGoods, ItemKindTorrent}, Summary: "内容形态。"},
		{Path: "url", Type: "string", Nullable: true, Required: true, Summary: "详情页链接。平台没有网页详情时为 null。"},
		{Path: "title", Type: "string", Nullable: true, Required: true, Summary: "标题。抖音作品只有描述、放在 text，title 恒为 null。"},
		{Path: "text", Type: "string", Nullable: true, Required: true, Summary: "正文或描述。"},
		{Path: "author", Type: "object", Nullable: true, Required: true, Summary: "user_ref。作者取不到时（例如 YouTube 的 Shorts 列表）为 null。"},
		{Path: "created_at", Type: "string", Nullable: true, Required: true, Summary: "发布时间，ISO 8601 带时区。列表里的相对时间按显示值推算，item get 才是精确值。"},
		{Path: "cover", Type: "string", Nullable: true, Required: true, Summary: "封面地址。取第一张图的地址；没有图时为 null。"},
		{Path: "media", Type: "array", Required: true, Summary: "media[]，按平台顺序。没有媒体时是 []。"},
		{Path: "stats", Type: "object", Required: true, Summary: "计数容器。"},
		{Path: "stats.views", Type: "integer", Nullable: true, Required: true, Summary: "播放 / 浏览数。抖音和小红书的公开列表接口不返回，恒为 null。"},
		{Path: "stats.likes", Type: "integer", Nullable: true, Required: true, Summary: "点赞数。"},
		{Path: "stats.comments", Type: "integer", Nullable: true, Required: true, Summary: "评论数。"},
		{Path: "stats.collects", Type: "integer", Nullable: true, Required: true, Summary: "收藏数。闲鱼是「想要」数。"},
		{Path: "stats.shares", Type: "integer", Nullable: true, Required: true, Summary: "分享 / 转发数。"},
		{Path: "price", Type: "object", Nullable: true, Required: true, Summary: "price。电商商品才有，其余为 null。"},
		{Path: "status", Type: "string", Nullable: true, Required: true, Values: []string{ItemStatusPublished, ItemStatusReviewing, ItemStatusRejected, ItemStatusPrivate, ItemStatusDraft, ItemStatusOnSale, ItemStatusSold, ItemStatusOffShelf}, Summary: "状态。只有自己的内容才带。"},
	},
	Notes: []string{
		"笔记、作品、稿件、推文、商品、种子全叫 item，形态用 kind 区分。",
		"字段恒为 null 的例子：抖音作品的 title、抖音和小红书列表的 stats.views。完整清单见各平台说明的「已知限制」。",
	},
}

var live = Object{
	Kind: KindLive,
	Fields: []Field{
		{Path: "id", Type: "string", Required: true, Summary: "直播间 ID / 房间号。"},
		{Path: "url", Type: "string", Nullable: true, Required: true, Summary: "直播间链接。"},
		{Path: "title", Type: "string", Nullable: true, Required: true, Summary: "直播标题。"},
		{Path: "status", Type: "string", Required: true, Values: []string{LiveStatusLive, LiveStatusOffline}, Summary: "直播状态。"},
		{Path: "host", Type: "object", Nullable: true, Required: true, Summary: "主播的 user_ref。"},
		{Path: "cover", Type: "string", Nullable: true, Required: true, Summary: "封面地址。"},
		{Path: "stats", Type: "object", Required: true, Summary: "计数容器。"},
		{Path: "stats.viewers", Type: "integer", Nullable: true, Required: true, Summary: "在线人数。平台不返回时为 null。"},
	},
}

var event = Object{
	Kind: KindEvent,
	Fields: []Field{
		{Path: "type", Type: "string", Required: true, Values: []string{EventTypeChat, EventTypeGift, EventTypeLike, EventTypeEnter, EventTypeFollow, EventTypeOther}, Summary: "事件类型。弹幕是 chat。"},
		{Path: "time", Type: "string", Required: true, Summary: "事件时间。平台不给时取本地收到的时刻。"},
		{Path: "user", Type: "object", Nullable: true, Required: true, Summary: "触发事件的用户的 user_ref。系统消息为 null。"},
		{Path: "text", Type: "string", Nullable: true, Required: true, Summary: "文字内容。gift / enter 等没有文字时为 null；点赞事件里是点赞数。"},
		{Path: "gift", Type: "object", Nullable: true, Required: true, Summary: "礼物：{name string, count integer}。非 gift 事件为 null。"},
		{Path: "gift.name", Type: "string", Required: true, Summary: "礼物名。"},
		{Path: "gift.count", Type: "integer", Required: true, Summary: "连击 / 数量。"},
	},
	Notes: []string{
		"live listen 一直输出事件，一行一个，直到 Ctrl-C 或 --duration 到期。",
	},
}

var gift = Object{
	Kind: KindGift,
	Fields: []Field{
		{Path: "id", Type: "string", Required: true, Summary: "礼物 ID。"},
		{Path: "name", Type: "string", Required: true, Summary: "礼物名。"},
		{Path: "price", Type: "object", Nullable: true, Required: true, Summary: "单价。"},
	},
}

var rank = Object{
	Kind: KindRank,
	Fields: []Field{
		{Path: "rank", Type: "integer", Required: true, Summary: "名次，从 1 开始。"},
		{Path: "user", Type: "object", Required: true, Summary: "上榜用户的 user_ref。"},
		{Path: "score", Type: "number", Nullable: true, Required: true, Summary: "分值。平台只排名不给分时为 null。"},
	},
}

var comment = Object{
	Kind: KindComment,
	Fields: []Field{
		{Path: "id", Type: "string", Required: true, Summary: "评论 ID。"},
		{Path: "item_id", Type: "string", Required: true, Summary: "所属内容 ID。"},
		{Path: "parent_id", Type: "string", Nullable: true, Required: true, Summary: "直接被回复的评论 ID。一级评论为 null。B 站的楼中楼可能指向另一条子回复。"},
		{Path: "author", Type: "object", Nullable: true, Required: true, Summary: "作者的 user_ref。评论已注销时为 null。"},
		{Path: "text", Type: "string", Required: true, Summary: "评论正文。"},
		{Path: "created_at", Type: "string", Nullable: true, Required: true, Summary: "发表时间。"},
		{Path: "stats", Type: "object", Required: true, Summary: "计数容器。"},
		{Path: "stats.likes", Type: "integer", Nullable: true, Required: true, Summary: "点赞数。"},
		{Path: "stats.replies", Type: "integer", Nullable: true, Required: true, Summary: "回复数。"},
	},
}

var conversation = Object{
	Kind: KindConversation,
	Fields: []Field{
		{Path: "id", Type: "string", Required: true, Summary: "会话 ID。小红书群聊是 group:<群 id>，淘宝是 <我的 id>.1-<对方 id>.1#11001，京东是 venderId，M-Team 是站内信 ID。"},
		{Path: "peer", Type: "object", Nullable: true, Required: true, Summary: "对方的 user_ref。群聊和系统通知为 null。"},
		{Path: "unread", Type: "integer", Nullable: true, Required: true, Summary: "未读数。平台不返回时为 null。"},
		{Path: "last_message", Type: "string", Nullable: true, Required: true, Summary: "最后一条消息的摘要。空会话为 null。"},
		{Path: "updated_at", Type: "string", Nullable: true, Required: true, Summary: "最后活跃时间。"},
	},
	Notes: []string{
		"会话 id 是不透明字符串，原样传回 msg history / msg send 即可。",
	},
}

var message = Object{
	Kind: KindMessage,
	Fields: []Field{
		{Path: "id", Type: "string", Required: true, Summary: "消息 ID。"},
		{Path: "conversation_id", Type: "string", Required: true, Summary: "所属会话 ID。"},
		{Path: "from", Type: "object", Nullable: true, Required: true, Summary: "发送者的 user_ref。系统消息为 null。", MessageField: true},
		{Path: "type", Type: "string", Required: true, Values: []string{MessageTypeText, MessageTypeImage, MessageTypeVideo, MessageTypeCard, MessageTypeOther}, Summary: "内容形态。交易卡片、分享卡片是 card；识别不出的是 other。", MessageField: true},
		{Path: "text", Type: "string", Nullable: true, Required: true, Summary: "文字内容。图片消息、纯卡片消息为 null；卡片会给摘要。", MessageField: true},
		{Path: "media", Type: "array", Required: true, Summary: "media[]。文字消息是 []。", MessageField: true},
		{Path: "created_at", Type: "string", Nullable: true, Required: true, Summary: "发出时间。平台不给时为 null。", MessageField: true},
	},
	Notes: []string{
		"闲鱼、淘宝的消息是 contentType 数字：1 文字、2 图片，其余按 card 处理并取摘要。",
		"X 的私信端到端加密，msg history 只给占位消息（type 为 other），原文用 --raw 看。",
		"抖音发文件时 type 为 other，只是文件名放在 text；发分享卡片时 type 为 card。",
	},
}

var notice = Object{
	Kind: KindNotice,
	Fields: []Field{
		{Path: "id", Type: "string", Required: true, Summary: "通知 ID。"},
		{Path: "type", Type: "string", Required: true, Values: []string{NoticeTypeComment, NoticeTypeMention, NoticeTypeLike, NoticeTypeFollow, NoticeTypeSystem}, Summary: "通知类型。"},
		{Path: "user", Type: "object", Nullable: true, Required: true, Summary: "触发者的 user_ref。系统通知为 null。"},
		{Path: "target", Type: "object", Nullable: true, Required: true, Summary: "被操作的对象：{id string, url string|null}。不挂对象的通知为 null。"},
		{Path: "target.id", Type: "string", Required: true, Summary: "对象 ID。"},
		{Path: "target.url", Type: "string", Nullable: true, Required: true, Summary: "对象链接。"},
		{Path: "text", Type: "string", Nullable: true, Required: true, Summary: "通知正文。模板通知取标题 + 正文。"},
		{Path: "created_at", Type: "string", Nullable: true, Required: true, Summary: "发生时间。"},
	},
}

var noticeCount = Object{
	Kind: "notice_count",
	Fields: []Field{
		{Path: "total", Type: "integer", Required: true, Summary: "未读总数。分类字段全为 null 时就是平台给的唯一数字。"},
		{Path: "comment", Type: "integer", Nullable: true, Required: true, Summary: "评论类未读数。"},
		{Path: "mention", Type: "integer", Nullable: true, Required: true, Summary: "@ 未读数。"},
		{Path: "like", Type: "integer", Nullable: true, Required: true, Summary: "点赞未读数。"},
		{Path: "follow", Type: "integer", Nullable: true, Required: true, Summary: "关注未读数。"},
		{Path: "system", Type: "integer", Nullable: true, Required: true, Summary: "系统通知未读数。"},
	},
	Notes: []string{
		"分类全为 null 时 total 是平台给的唯一数字（快手只有 unReadCount，YouTube 只有未读总数）。",
	},
}

var authStatus = Object{
	Kind: KindAuthStatus,
	Fields: []Field{
		{Path: "logged_in", Type: "boolean", Required: true, Summary: "是否登录。"},
		{Path: "user", Type: "object", Nullable: true, Required: true, Summary: "当前账号的 user_ref。未登录或平台不返回账号时为 null。"},
		{Path: "method", Type: "string", Nullable: true, Required: true, Summary: "当初的登录方式：qrcode / sms / password / cookie / oauth / apikey / guest。"},
		{Path: "expires_at", Type: "string", Nullable: true, Required: true, Summary: "登录态过期时间。平台不给时为 null。"},
	},
}

var account = Object{
	Kind: KindAccount,
	Fields: []Field{
		{Path: "platform", Type: "string", Required: true, Summary: "平台规范 id。"},
		{Path: "endpoint", Type: "string", Required: true, Summary: "端：web / app / pc。"},
		{Path: "account", Type: "string", Required: true, Summary: "账号名。"},
		{Path: "current", Type: "boolean", Required: true, Summary: "是否是当前账号。"},
		{Path: "user", Type: "object", Nullable: true, Required: true, Summary: "账号对应的用户 user_ref。"},
		{Path: "method", Type: "string", Nullable: true, Required: true, Summary: "登录方式。"},
		{Path: "updated_at", Type: "string", Nullable: true, Required: true, Summary: "最后更新时间。"},
	},
}

var file = Object{
	Kind: KindFile,
	Fields: []Field{
		{Path: "path", Type: "string", Required: true, Summary: "本地文件路径。"},
		{Path: "type", Type: "string", Required: true, Values: []string{"image", "video", "audio", "torrent"}, Summary: "文件类型。"},
		{Path: "url", Type: "string", Nullable: true, Required: true, Summary: "临时下载地址。M-Team 的种子链接含个人 passkey，不输出。"},
		{Path: "size", Type: "integer", Required: true, Summary: "字节数。"},
	},
}

// Elements 是入站消息正文的元素契约，顺序有意义。
var Elements = map[string]Object{
	ElementText: {
		Kind:   ElementText,
		Fields: []Field{{Path: "text", Type: "string", Required: true, Summary: "文字。"}},
	},
	ElementImage: {
		Kind:   ElementImage,
		Fields: media.Fields,
	},
	ElementAudio: {
		Kind:   ElementAudio,
		Fields: media.Fields,
	},
	ElementVideo: {
		Kind: ElementVideo,
		Fields: append([]Field{
			{Path: "thumbnail", Type: "string", Nullable: true, Required: true, Summary: "缩略图地址。"},
		}, media.Fields...),
	},
	ElementFile: {
		Kind: ElementFile,
		Fields: []Field{
			{Path: "url", Type: "string", Required: true, Summary: "文件地址。"},
			{Path: "name", Type: "string", Required: true, Summary: "文件名。"},
			{Path: "size", Type: "integer", Nullable: true, Required: true, Summary: "字节数。"},
			{Path: "mime_type", Type: "string", Nullable: true, Required: true, Summary: "MIME 类型。"},
		},
	},
	ElementProductCard: {
		Kind: ElementProductCard,
		Fields: []Field{
			{Path: "item_id", Type: "string", Required: true, Summary: "商品 ID。"},
			{Path: "title", Type: "string", Required: true, Summary: "商品标题。"},
			{Path: "price", Type: "number", Required: true, Summary: "价格，主单位。"},
			{Path: "image_url", Type: "string", Required: true, Summary: "商品图。"},
			{Path: "detail_url", Type: "string", Required: true, Summary: "商品链接。"},
			{Path: "platform", Type: "string", Nullable: true, Required: true, Summary: "来源平台。"},
			{Path: "sku", Type: "string", Nullable: true, Required: true, Summary: "SKU。"},
			{Path: "stock", Type: "integer", Nullable: true, Required: true, Summary: "库存。"},
		},
	},
	ElementOrderInfo: {
		Kind: ElementOrderInfo,
		Fields: []Field{
			{Path: "order_id", Type: "string", Required: true, Summary: "订单号。"},
			{Path: "status", Type: "string", Required: true, Values: []string{"pending", "paid", "shipped", "delivered", "cancelled", "refunded"}, Summary: "订单状态。"},
			{Path: "amount", Type: "number", Required: true, Summary: "金额。"},
			{Path: "currency", Type: "string", Nullable: true, Required: true, Summary: "币种。"},
			{Path: "item_count", Type: "integer", Nullable: true, Required: true, Summary: "件数。"},
			{Path: "created_at", Type: "integer", Nullable: true, Required: true, Summary: "下单时间（毫秒）。"},
			{Path: "paid_at", Type: "integer", Nullable: true, Required: true, Summary: "付款时间（毫秒）。"},
			{Path: "shipped_at", Type: "integer", Nullable: true, Required: true, Summary: "发货时间（毫秒）。"},
			{Path: "delivered_at", Type: "integer", Nullable: true, Required: true, Summary: "送达时间（毫秒）。"},
		},
	},
	ElementInquiry: {
		Kind: ElementInquiry,
		Fields: []Field{
			{Path: "product_id", Type: "string", Nullable: true, Required: true, Summary: "咨询的商品。"},
			{Path: "order_id", Type: "string", Nullable: true, Required: true, Summary: "咨询的订单。"},
			{Path: "question", Type: "string", Required: true, Summary: "问题原文。"},
			{Path: "category", Type: "string", Nullable: true, Required: true, Values: []string{"price", "shipping", "quality", "return", "other"}, Summary: "问题分类。"},
		},
	},
	ElementLocation: {
		Kind: ElementLocation,
		Fields: []Field{
			{Path: "latitude", Type: "number", Required: true, Summary: "纬度。"},
			{Path: "longitude", Type: "number", Required: true, Summary: "经度。"},
			{Path: "address", Type: "string", Nullable: true, Required: true, Summary: "地址。"},
			{Path: "name", Type: "string", Nullable: true, Required: true, Summary: "地点名。"},
		},
	},
	ElementEmoji: {
		Kind: ElementEmoji,
		Fields: []Field{
			{Path: "emoji", Type: "string", Required: true, Summary: "表情字符。"},
			{Path: "name", Type: "string", Nullable: true, Required: true, Summary: "表情名。"},
		},
	},
	ElementMention: {
		Kind: ElementMention,
		Fields: []Field{
			{Path: "user_id", Type: "string", Required: true, Summary: "被 @ 的用户 ID。"},
			{Path: "username", Type: "string", Required: true, Summary: "被 @ 的用户名。"},
		},
	},
}

// catalog 是全部归一化对象。
var catalog = map[string]*Object{
	KindUser:         &user,
	"user_ref":       &userRef,
	KindItem:         &item,
	KindMedia:        &media,
	KindPrice:        &price,
	KindComment:      &comment,
	KindLive:         &live,
	KindEvent:        &event,
	KindGift:         &gift,
	KindRank:         &rank,
	KindConversation: &conversation,
	KindMessage:      &message,
	KindNotice:       &notice,
	"notice_count":   &noticeCount,
	KindAuthStatus:   &authStatus,
	KindAccount:      &account,
	KindFile:         &file,
}

// Objects 返回全部归一化对象。
func Objects() map[string]*Object {
	return catalog
}

// ObjectOf 按类型名取对象。
func ObjectOf(kind string) (*Object, bool) {
	o, ok := catalog[kind]
	return o, ok
}

// AllFields 返回到全部字段契约的扁平视图，键是 `<kind>.<path>`。
func AllFields() map[string]Field {
	out := map[string]Field{}
	for kind, o := range catalog {
		for _, f := range o.Fields {
			out[kind+"."+f.Path] = f
		}
	}
	return out
}

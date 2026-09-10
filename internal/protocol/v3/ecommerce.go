package v3

import (
	"encoding/json"
	"time"
)

// EcommerceMessage represents an e-commerce specific message
type EcommerceMessage struct {
	MessageChain *MessageChain     `json:"message_chain"`
	ProductCards []ProductCard     `json:"product_cards,omitempty"`
	OrderInfo    *OrderInfo        `json:"order_info,omitempty"`
	Inquiry      *Inquiry          `json:"inquiry,omitempty"`
	Context      *EcommerceContext `json:"context,omitempty"`
}

// EcommerceContext represents e-commerce conversation context
type EcommerceContext struct {
	ConversationType string `json:"conversation_type"` // pre_sale, after_sale, inquiry, complaint
	ProductID        string `json:"product_id,omitempty"`
	OrderID          string `json:"order_id,omitempty"`
	CustomerLevel    string `json:"customer_level,omitempty"` // vip, regular, new
	PreviousOrders   int    `json:"previous_orders,omitempty"`
}

// ProductDetail represents detailed product information
type ProductDetail struct {
	ProductCard
	Description string   `json:"description,omitempty"`
	Specs       []Spec   `json:"specs,omitempty"`
	Images      []string `json:"images,omitempty"`
	Category    string   `json:"category,omitempty"`
	Brand       string   `json:"brand,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Rating      float64  `json:"rating,omitempty"`
	ReviewCount int      `json:"review_count,omitempty"`
}

// Spec represents product specification
type Spec struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// OrderDetail represents detailed order information
type OrderDetail struct {
	OrderInfo
	Items      []OrderItem `json:"items,omitempty"`
	Shipping   *Shipping   `json:"shipping,omitempty"`
	Payment    *Payment    `json:"payment,omitempty"`
	BuyerInfo  *UserInfo   `json:"buyer_info,omitempty"`
	SellerInfo *UserInfo   `json:"seller_info,omitempty"`
}

// OrderItem represents an order item
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Title     string  `json:"title"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	ImageURL  string  `json:"image_url,omitempty"`
	SKU       string  `json:"sku,omitempty"`
}

// Shipping represents shipping information
type Shipping struct {
	Method      string  `json:"method"` // express, standard, pickup
	TrackingNo  string  `json:"tracking_no,omitempty"`
	Carrier     string  `json:"carrier,omitempty"`
	EstimatedAt int64   `json:"estimated_at,omitempty"`
	ShippedAt   int64   `json:"shipped_at,omitempty"`
	DeliveredAt int64   `json:"delivered_at,omitempty"`
	Address     string  `json:"address,omitempty"`
	Fee         float64 `json:"fee,omitempty"`
}

// Payment represents payment information
type Payment struct {
	Method        string  `json:"method"` // alipay, wechat, credit_card
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"` // pending, paid, refunded
	PaidAt        int64   `json:"paid_at,omitempty"`
	TransactionID string  `json:"transaction_id,omitempty"`
}

// UserInfo represents user information
type UserInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Phone   string `json:"phone,omitempty"`
	Address string `json:"address,omitempty"`
	Level   string `json:"level,omitempty"`
}

// InquiryDetail represents detailed inquiry information
type InquiryDetail struct {
	Inquiry
	ProductDetail *ProductDetail `json:"product_detail,omitempty"`
	OrderDetail   *OrderDetail   `json:"order_detail,omitempty"`
	Urgency       string         `json:"urgency,omitempty"` // low, medium, high
	AssignedTo    string         `json:"assigned_to,omitempty"`
	Status        string         `json:"status,omitempty"` // open, in_progress, resolved, closed
}

// NewEcommerceMessage creates a new e-commerce message
func NewEcommerceMessage(platform, instance string, sender SenderInfo) *EcommerceMessage {
	return &EcommerceMessage{
		MessageChain: NewMessageChain(platform, instance, sender),
		ProductCards: make([]ProductCard, 0),
	}
}

// AddProduct adds a product card to the message
func (em *EcommerceMessage) AddProduct(card ProductCard) *EcommerceMessage {
	em.ProductCards = append(em.ProductCards, card)
	em.MessageChain.AddProductCard(card)
	return em
}

// SetOrderInfo sets order information
func (em *EcommerceMessage) SetOrderInfo(order OrderInfo) *EcommerceMessage {
	em.OrderInfo = &order
	em.MessageChain.AddOrderInfo(order)
	return em
}

// SetInquiry sets inquiry information
func (em *EcommerceMessage) SetInquiry(inquiry Inquiry) *EcommerceMessage {
	em.Inquiry = &inquiry
	em.MessageChain.AddInquiry(inquiry)
	return em
}

// SetContext sets conversation context
func (em *EcommerceMessage) SetContext(ctx *EcommerceContext) *EcommerceMessage {
	em.Context = ctx
	return em
}

// ToJSON converts to JSON
func (em *EcommerceMessage) ToJSON() ([]byte, error) {
	return json.Marshal(em)
}

// EcommerceMessageFromJSON parses JSON
func EcommerceMessageFromJSON(data []byte) (*EcommerceMessage, error) {
	var em EcommerceMessage
	if err := json.Unmarshal(data, &em); err != nil {
		return nil, err
	}
	return &em, nil
}

// OrderStatus constants
const (
	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusShipped   = "shipped"
	OrderStatusDelivered = "delivered"
	OrderStatusCancelled = "cancelled"
	OrderStatusRefunded  = "refunded"
)

// LogisticsStatus constants (发货/物流状态).
const (
	LogisticsStatusPending   = "pending"    // 待发货
	LogisticsStatusShipped   = "shipped"    // 已发货
	LogisticsStatusInTransit = "in_transit" // 运输中
	LogisticsStatusDelivered = "delivered"  // 已签收
	LogisticsStatusException = "exception"  // 异常
)

// ShippingMethod constants
const (
	ShippingMethodExpress  = "express"
	ShippingMethodStandard = "standard"
	ShippingMethodPickup   = "pickup"
)

// ActivityType constants (活动/事件类型).
const (
	ActivityTypePromotion = "promotion"
	ActivityTypeFlashSale = "flash_sale"
	ActivityTypeCampaign  = "campaign"
	ActivityTypeCoupon    = "coupon"
)

// ConversationType constants
const (
	ConversationTypePreSale   = "pre_sale"
	ConversationTypeAfterSale = "after_sale"
	ConversationTypeInquiry   = "inquiry"
	ConversationTypeComplaint = "complaint"
)

// CustomerLevel constants
const (
	CustomerLevelVIP     = "vip"
	CustomerLevelRegular = "regular"
	CustomerLevelNew     = "new"
)

// InquiryCategory constants
const (
	InquiryCategoryPrice    = "price"
	InquiryCategoryShipping = "shipping"
	InquiryCategoryQuality  = "quality"
	InquiryCategoryReturn   = "return"
	InquiryCategoryOther    = "other"
)

// Urgency constants
const (
	UrgencyLow    = "low"
	UrgencyMedium = "medium"
	UrgencyHigh   = "high"
)

// InquiryStatus constants
const (
	InquiryStatusOpen       = "open"
	InquiryStatusInProgress = "in_progress"
	InquiryStatusResolved   = "resolved"
	InquiryStatusClosed     = "closed"
)

// ActivityEvent is the full activity/event information envelope used when an
// adapter reports "活动信息接收" (activity.received) to downstream frameworks.
type ActivityEvent struct {
	EventType      string                 `json:"event_type"`
	Platform       string                 `json:"platform,omitempty"`
	InstanceID     string                 `json:"instance_id,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	ActivityID     string                 `json:"activity_id,omitempty"`
	ProductID      string                 `json:"product_id,omitempty"`
	OrderID        string                 `json:"order_id,omitempty"`
	Timestamp      int64                  `json:"timestamp"`
	Data           map[string]interface{} `json:"data,omitempty"`
}

// EcommerceEvent is a generic e-commerce lifecycle event (order.*, logistics
// updated, product updated, activity received). It lets adapters push active
// information to downstream frameworks without inventing ad-hoc payloads.
type EcommerceEvent struct {
	EventType  EventType              `json:"event_type"`
	Platform   string                 `json:"platform,omitempty"`
	InstanceID string                 `json:"instance_id,omitempty"`
	OrderID    string                 `json:"order_id,omitempty"`
	ProductID  string                 `json:"product_id,omitempty"`
	Timestamp  int64                  `json:"timestamp"`
	Order      *OrderDetail           `json:"order,omitempty"`
	Logistics  *LogisticsInfo         `json:"logistics,omitempty"`
	Product    *ProductDetail         `json:"product,omitempty"`
	Activity   *ActivityEvent         `json:"activity,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

// CreateProductCard creates a product card from product detail
func CreateProductCard(detail ProductDetail) ProductCard {
	return ProductCard{
		ItemID:        detail.ItemID,
		Title:         detail.Title,
		Price:         detail.Price,
		ImageURL:      detail.ImageURL,
		DetailURL:     detail.DetailURL,
		Platform:      detail.Platform,
		SKU:           detail.SKU,
		Stock:         detail.Stock,
		Description:   detail.Description,
		Images:        detail.Images,
		OriginalPrice: detail.OriginalPrice,
		Currency:      detail.Currency,
		Sales:         detail.Sales,
		Category:      detail.Category,
		Brand:         detail.Brand,
		Tags:          detail.Tags,
		Rating:        detail.Rating,
		ReviewCount:   detail.ReviewCount,
	}
}

// CreateLogisticsInfo builds a LogisticsInfo from a Shipping record and an
// optional trace timeline.
func CreateLogisticsInfo(orderID string, shipping *Shipping, traces []LogisticsTrace) LogisticsInfo {
	info := LogisticsInfo{
		OrderID:   orderID,
		Traces:    traces,
		UpdatedAt: GetCurrentTimestamp(),
	}
	if shipping != nil {
		info.TrackingNo = shipping.TrackingNo
		info.Carrier = shipping.Carrier
		info.EstimatedAt = shipping.EstimatedAt
		info.ShippedAt = shipping.ShippedAt
		info.DeliveredAt = shipping.DeliveredAt
		switch {
		case shipping.DeliveredAt > 0:
			info.Status = LogisticsStatusDelivered
		case shipping.ShippedAt > 0:
			info.Status = LogisticsStatusShipped
		default:
			info.Status = LogisticsStatusPending
		}
	}
	return info
}

// CreateOrderInfo creates order info from order detail
func CreateOrderInfo(detail OrderDetail) OrderInfo {
	return OrderInfo{
		OrderID:     detail.OrderID,
		Status:      detail.Status,
		Amount:      detail.Amount,
		Currency:    detail.Currency,
		ItemCount:   detail.ItemCount,
		CreatedAt:   detail.CreatedAt,
		PaidAt:      detail.PaidAt,
		ShippedAt:   detail.ShippedAt,
		DeliveredAt: detail.DeliveredAt,
	}
}

// GetCurrentTimestamp returns current timestamp in milliseconds
func GetCurrentTimestamp() int64 {
	return time.Now().UnixMilli()
}

// ─────────────────────────────────────────────────────────────────────────────
// XianYuApis 原版兼容字段（goofish_apis.py / goofish_live.py / message.types）
//
// 以下结构体与常量 1:1 对齐 XianYuApis 原版的请求/响应字段，确保 ESPL2 的
// "非活动字段" 机制可以完整承载原版全部可调用接口与字段。
// ─────────────────────────────────────────────────────────────────────────────

// ItemStatus constants (商品状态).
const (
	ItemStatusOnSale   = "on_sale"   // 在售
	ItemStatusOffShelf = "off_shelf" // 已下架
	ItemStatusSoldOut  = "sold_out"  // 已售出
	ItemStatusDraft    = "draft"     // 草稿
)

// ContentType constants mirror XianYuApis message.types content type numbers.
// They are the wire-level `content.contentType` values of sendByReceiverScope.
const (
	ContentTypeText      = "1"  // 文本
	ContentTypeImage     = "2"  // 图片
	ContentTypeAudio     = "3"  // 音频
	ContentTypeVideo     = "4"  // 视频
	ContentTypeTradeCard = "26" // 交易卡片（dxCard）
)

// DeliveryChoice constants mirror XianYuApis message.types.DeliverySettings.choice.
const (
	DeliveryChoiceFreeShipping = "包邮"    // 包邮
	DeliveryChoiceByDistance   = "按距离计费" // 按距离计费
	DeliveryChoiceFixedPrice   = "一口价"   // 一口价
	DeliveryChoiceNoDelivery   = "无需邮寄"  // 无需邮寄
)

// TradeCardRole constants for dxCard targetUrl role.
const (
	TradeCardRoleSeller = "seller"
	TradeCardRoleBuyer  = "buyer"
)

// PublishScene constants (publishScene).
const (
	PublishSceneDefault = "default"
	PublishSceneBulk    = "bulk"
)

// QRCodeLoginStatus constants.
const (
	QRLoginStatusPending   = "pending"   // 等待扫码
	QRLoginStatusScanned   = "scanned"   // 已扫码待确认
	QRLoginStatusConfirmed = "confirmed" // 已确认
	QRLoginStatusExpired   = "expired"   // 已过期
)

// MediaType constants for upload_media.
const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
	MediaTypeAudio = "audio"
)

// ── 商品发布字段（mtop.idle.pc.idleitem.publish）───────────────────────────

// Price mirrors XianYuApis message.types.Price.
type Price struct {
	CurrentPrice  float64 `json:"current_price"`
	OriginalPrice float64 `json:"original_price,omitempty"`
	Currency      string  `json:"currency,omitempty"`
	PriceInCent   int64   `json:"price_in_cent,omitempty"`
	OrigPriceCent int64   `json:"orig_price_in_cent,omitempty"`
}

// DeliverySettings mirrors XianYuApis message.types.DeliverySettings.
type DeliverySettings struct {
	Choice          string  `json:"choice"` // 包邮 / 按距离计费 / 一口价 / 无需邮寄
	PostPrice       float64 `json:"post_price,omitempty"`
	PostPriceInCent int64   `json:"post_price_in_cent,omitempty"`
	CanSelfPickup   bool    `json:"can_self_pickup,omitempty"`
}

// ImageInfoDO mirrors XianYuApis imageInfoDOList[] entry.
type ImageInfoDO struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Pix    string `json:"pix,omitempty"`
}

// ItemTextDTO mirrors XianYuApis itemTextDTO.
type ItemTextDTO struct {
	Desc              string `json:"desc"`
	Title             string `json:"title"`
	TitleDescSeparate bool   `json:"titleDescSeparate,omitempty"`
}

// ItemPriceDTO mirrors XianYuApis itemPriceDTO.
type ItemPriceDTO struct {
	PriceInCent     int64 `json:"priceInCent"`
	OrigPriceInCent int64 `json:"origPriceInCent,omitempty"`
}

// ItemPostFeeDTO mirrors XianYuApis itemPostFeeDTO.
type ItemPostFeeDTO struct {
	CanFreeShipping bool   `json:"canFreeShipping,omitempty"`
	SupportFreight  bool   `json:"supportFreight,omitempty"`
	OnlyTakeSelf    bool   `json:"onlyTakeSelf,omitempty"`
	TemplateID      string `json:"templateId,omitempty"`
	PostPriceInCent int64  `json:"postPriceInCent,omitempty"`
}

// ItemCatDTO mirrors XianYuApis itemCatDTO.
type ItemCatDTO struct {
	CatID        string `json:"catId,omitempty"`
	CatName      string `json:"catName,omitempty"`
	ChannelCatID string `json:"channelCatId,omitempty"`
	TbCatID      string `json:"tbCatId,omitempty"`
}

// ItemAddrDTO mirrors XianYuApis itemAddrDTO.
type ItemAddrDTO struct {
	Area       string `json:"area,omitempty"`
	City       string `json:"city,omitempty"`
	DivisionID string `json:"divisionId,omitempty"`
	Gps        string `json:"gps,omitempty"`
	PoiID      string `json:"poiId,omitempty"`
	PoiName    string `json:"poiName,omitempty"`
	Prov       string `json:"prov,omitempty"`
}

// ItemLabelExt mirrors XianYuApis itemLabelExtList[] entry.
type ItemLabelExt struct {
	LabelID   string `json:"labelId,omitempty"`
	LabelName string `json:"labelName,omitempty"`
	LabelType string `json:"labelType,omitempty"`
	Value     string `json:"value,omitempty"`
}

// UserRightsProtocol mirrors XianYuApis userRightsProtocols[] entry.
type UserRightsProtocol struct {
	Enable      bool   `json:"enable"`
	ServiceCode string `json:"serviceCode,omitempty"`
}

// ItemPublish is the full publish payload mirroring XianYuApis `public()`.
type ItemPublish struct {
	Freebies            bool                 `json:"freebies,omitempty"`
	ItemTypeStr         string               `json:"itemTypeStr,omitempty"` // 默认 "b"
	Quantity            int                  `json:"quantity,omitempty"`
	SimpleItem          bool                 `json:"simpleItem,omitempty"`
	ImageInfoDOList     []ImageInfoDO        `json:"imageInfoDOList,omitempty"`
	ItemTextDTO         *ItemTextDTO         `json:"itemTextDTO,omitempty"`
	ItemLabelExtList    []ItemLabelExt       `json:"itemLabelExtList,omitempty"`
	ItemPriceDTO        *ItemPriceDTO        `json:"itemPriceDTO,omitempty"`
	UserRightsProtocols []UserRightsProtocol `json:"userRightsProtocols,omitempty"`
	ItemPostFeeDTO      *ItemPostFeeDTO      `json:"itemPostFeeDTO,omitempty"`
	ItemAddrDTO         *ItemAddrDTO         `json:"itemAddrDTO,omitempty"`
	DefaultPrice        string               `json:"defaultPrice,omitempty"`
	ItemCatDTO          *ItemCatDTO          `json:"itemCatDTO,omitempty"`
	UniqueCode          string               `json:"uniqueCode,omitempty"`
	SourceID            string               `json:"sourceId,omitempty"`
	Bizcode             string               `json:"bizcode,omitempty"`
	PublishScene        string               `json:"publishScene,omitempty"`
	Delivery            *DeliverySettings    `json:"delivery,omitempty"`
}

// ItemPublishResult is the response of publish_item.
type ItemPublishResult struct {
	ItemID       string `json:"item_id"`
	Status       string `json:"status"`
	PublishScene string `json:"publish_scene,omitempty"`
	CreatedAt    int64  `json:"created_at,omitempty"`
}

// ItemUpdateResult is the response of update_item / update_item_price.
type ItemUpdateResult struct {
	ItemID    string `json:"item_id"`
	Status    string `json:"status,omitempty"`
	Price     *Price `json:"price,omitempty"`
	UpdatedAt int64  `json:"updated_at"`
}

// ── 分类推荐 / 地址（mtop.taobao.idle.kgraph.property.recommend）─────────

// CardListItem mirrors XianYuApis get_public_channel cardList entry.
type CardListItem struct {
	CardID   string                 `json:"card_id,omitempty"`
	Title    string                 `json:"title,omitempty"`
	Subtitle string                 `json:"subtitle,omitempty"`
	ImageURL string                 `json:"image_url,omitempty"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

// CategoryPredictResult mirrors XianYuApis categoryPredictResult.
type CategoryPredictResult struct {
	CatID        string `json:"cat_id,omitempty"`
	CatName      string `json:"cat_name,omitempty"`
	ChannelCatID string `json:"channel_cat_id,omitempty"`
	TbCatID      string `json:"tb_cat_id,omitempty"`
}

// CategoryRecommend is the response of get_category_recommend.
type CategoryRecommend struct {
	CardList              []CardListItem         `json:"card_list,omitempty"`
	CategoryPredictResult *CategoryPredictResult `json:"category_predict_result,omitempty"`
	Raw                   map[string]interface{} `json:"raw,omitempty"`
}

// DefaultAddress mirrors XianYuApis get_default_location commonAddresses entry.
type DefaultAddress struct {
	PoiID      string `json:"poi_id,omitempty"`
	PoiName    string `json:"poi_name,omitempty"`
	Area       string `json:"area,omitempty"`
	City       string `json:"city,omitempty"`
	Prov       string `json:"prov,omitempty"`
	DivisionID string `json:"division_id,omitempty"`
	Gps        string `json:"gps,omitempty"`
}

// ── 媒体上传 / 登录态（stream-upload.goofish.com/api/upload.api）─────────

// MediaUpload mirrors XianYuApis upload_media result `object`.
type MediaUpload struct {
	URL      string `json:"url"`
	Pix      string `json:"pix,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// TokenResult mirrors XianYuApis get_token / refresh_token response.
type TokenResult struct {
	Cookies     string `json:"cookies,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	RefreshedAt int64  `json:"refreshed_at,omitempty"`
}

// QRCodeLogin mirrors XianYuApis qrcode_login.
type QRCodeLogin struct {
	QRURL     string `json:"qr_url,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Status    string `json:"status,omitempty"` // pending / scanned / confirmed / expired
	Cookies   string `json:"cookies,omitempty"`
}

// ── 会话 / 消息（goofish_live.py lwp 命令）───────────────────────────────

// HistoryMessage is one entry of listUserMessages.userMessageModels.
type HistoryMessage struct {
	MessageID        string                 `json:"message_id,omitempty"`
	Cid              string                 `json:"cid,omitempty"`
	SenderID         string                 `json:"sender_id,omitempty"`
	ReceiverID       string                 `json:"receiver_id,omitempty"`
	ContentType      string                 `json:"content_type,omitempty"`
	Content          string                 `json:"content,omitempty"`
	Custom           map[string]interface{} `json:"custom,omitempty"`
	CreatedAt        int64                  `json:"created_at,omitempty"`
	ReadStatus       int                    `json:"read_status,omitempty"`
	ConversationType int                    `json:"conversation_type,omitempty"`
}

// ConversationHistory mirrors XianYuApis listUserMessages response.
type ConversationHistory struct {
	HasMore    bool             `json:"has_more"`
	NextCursor string           `json:"next_cursor,omitempty"`
	Messages   []HistoryMessage `json:"messages"`
}

// ConversationCreate mirrors XianYuApis SingleChatConversation.create.
type ConversationCreate struct {
	Cid              string `json:"cid"`
	ConversationType int    `json:"conversation_type,omitempty"`
	PairFirst        string `json:"pair_first,omitempty"`
	PairSecond       string `json:"pair_second,omitempty"`
	BizType          string `json:"biz_type,omitempty"`
	ItemID           string `json:"item_id,omitempty"`
	Ctx              string `json:"ctx,omitempty"`
	CreatedAt        int64  `json:"created_at,omitempty"`
}

// SendMessageResult mirrors XianYuApis sendByReceiverScope response.
type SendMessageResult struct {
	UUID   string `json:"uuid,omitempty"`
	MsgID  string `json:"msg_id,omitempty"`
	SentAt int64  `json:"sent_at,omitempty"`
}

// ── 交易卡片（dxCard，contentType=26）────────────────────────────────────

// TradeCardButton mirrors the dxCard button node.
type TradeCardButton struct {
	Text string `json:"text,omitempty"`
	URL  string `json:"url,omitempty"`
}

// TradeCardItem mirrors dxCard.item.main.
type TradeCardItem struct {
	ClickParam string                 `json:"click_param,omitempty"`
	ExContent  map[string]interface{} `json:"ex_content,omitempty"`
	TargetURL  string                 `json:"target_url,omitempty"` // fleamarket://order_detail?id=..&role=seller
}

// TradeCardTemplate mirrors dxCard.template.
type TradeCardTemplate struct {
	Name    string `json:"name,omitempty"` // idlefish_message_trade_chat_card
	URL     string `json:"url,omitempty"`
	Version string `json:"version,omitempty"`
}

// TradeCard mirrors the XianYuApis dxCard (contentType=26) structure, including
// extension.bizTag / detailNotice / extJson used by the reference impl.
type TradeCard struct {
	Item              *TradeCardItem         `json:"item,omitempty"`
	TargetURL         string                 `json:"target_url,omitempty"`
	Template          *TradeCardTemplate     `json:"template,omitempty"`
	Extension         map[string]interface{} `json:"extension,omitempty"`
	ClosePushReceiver bool                   `json:"close_push_receiver,omitempty"`
	CloseUnreadNumber bool                   `json:"close_unread_number,omitempty"`
	DetailNotice      string                 `json:"detail_notice,omitempty"`
	ExtJSON           map[string]interface{} `json:"ext_json,omitempty"`
	BgColor           string                 `json:"bg_color,omitempty"`
	Title             string                 `json:"title,omitempty"`
	Desc              string                 `json:"desc,omitempty"`
}

// ── 转换助手 ────────────────────────────────────────────────────────────

// CreatePrice builds a Price from cent values.
func CreatePrice(priceInCent, origPriceInCent int64, currency string) Price {
	if currency == "" {
		currency = "CNY"
	}
	return Price{
		CurrentPrice:  float64(priceInCent) / 100,
		OriginalPrice: float64(origPriceInCent) / 100,
		PriceInCent:   priceInCent,
		OrigPriceCent: origPriceInCent,
		Currency:      currency,
	}
}

// CreateDeliverySettings builds DeliverySettings from a choice and post fee.
func CreateDeliverySettings(choice string, postPriceInCent int64, canSelfPickup bool) DeliverySettings {
	return DeliverySettings{
		Choice:          choice,
		PostPrice:       float64(postPriceInCent) / 100,
		PostPriceInCent: postPriceInCent,
		CanSelfPickup:   canSelfPickup,
	}
}

// TradeCardURL builds the fleamarket order-detail deep link used by dxCard.
func TradeCardURL(orderID, role string) string {
	if role == "" {
		role = TradeCardRoleSeller
	}
	return "fleamarket://order_detail?id=" + orderID + "&role=" + role
}

// CreateTradeCard builds a minimal dxCard for an order.
func CreateTradeCard(orderID, role, title, desc string) TradeCard {
	return TradeCard{
		Item: &TradeCardItem{
			TargetURL: TradeCardURL(orderID, role),
		},
		TargetURL: TradeCardURL(orderID, role),
		Template: &TradeCardTemplate{
			Name: "idlefish_message_trade_chat_card",
		},
		Title: title,
		Desc:  desc,
	}
}

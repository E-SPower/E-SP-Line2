package v3

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"time"
)

// MessageChain represents a message chain for V3 protocol
type MessageChain struct {
	ID        string           `json:"id"`
	Timestamp int64            `json:"timestamp"`
	Platform  string           `json:"platform"`
	Instance  string           `json:"instance"`
	Sender    SenderInfo       `json:"sender"`
	Content   []MessageElement `json:"content"`
	Raw       json.RawMessage  `json:"raw,omitempty"`
	Hash      string           `json:"hash"`
}

// SenderInfo represents sender information
type SenderInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar,omitempty"`
	Platform string `json:"platform,omitempty"`
}

// MessageElement represents a message element in the chain
type MessageElement struct {
	Type    ElementType `json:"type"`
	Content interface{} `json:"content"`
}

// ElementType represents the type of message element
type ElementType string

const (
	ElementTypeText        ElementType = "text"
	ElementTypeImage       ElementType = "image"
	ElementTypeAudio       ElementType = "audio"
	ElementTypeVideo       ElementType = "video"
	ElementTypeFile        ElementType = "file"
	ElementTypeProductCard ElementType = "product_card"
	ElementTypeOrderInfo   ElementType = "order_info"
	ElementTypeLogistics   ElementType = "logistics"
	ElementTypeActivity    ElementType = "activity"
	ElementTypeInquiry     ElementType = "inquiry"
	ElementTypeLocation    ElementType = "location"
	ElementTypeEmoji       ElementType = "emoji"
	ElementTypeMention     ElementType = "mention"
	// ElementTypeTradeCard carries a XianYu dxCard (contentType=26) trade card,
	// matching XianYuApis message parsing for 交易卡片 messages.
	ElementTypeTradeCard ElementType = "trade_card"
)

// TextContent represents text message content
type TextContent struct {
	Text string `json:"text"`
}

// ImageContent represents image message content
type ImageContent struct {
	URL      string `json:"url"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// AudioContent represents audio message content. It carries both the ESPL2
// canonical fields (url/duration) and the XianYuApis wire field names
// (audio_url/duration_ms) so a bridge can round-trip the original payload.
type AudioContent struct {
	URL      string `json:"url"`
	Duration int    `json:"duration,omitempty"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`

	// XianYuApis message.types.AudioContent 兼容字段。
	AudioURL   string `json:"audio_url,omitempty"`
	DurationMS int    `json:"duration_ms,omitempty"`
}

// VideoContent represents video message content
type VideoContent struct {
	URL       string `json:"url"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Duration  int    `json:"duration,omitempty"`
	Size      int64  `json:"size,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// FileContent represents file message content
type FileContent struct {
	URL      string `json:"url"`
	Name     string `json:"name"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// ProductCard represents e-commerce product card. It carries both the fields
// needed for an inline card and the "inactive fields" (名称/简介/图片) that a
// downstream framework may actively query via the get_product tool.
type ProductCard struct {
	ItemID    string  `json:"item_id"`
	Title     string  `json:"title"`     // 商品名称
	Price     float64 `json:"price"`     // 当前价格
	ImageURL  string  `json:"image_url"` // 主图
	DetailURL string  `json:"detail_url"`
	Platform  string  `json:"platform,omitempty"`
	SKU       string  `json:"sku,omitempty"`
	Stock     int     `json:"stock,omitempty"`

	// ── 非活动字段（可主动查询）──
	Description   string   `json:"description,omitempty"`    // 商品简介
	Images        []string `json:"images,omitempty"`         // 图片列表
	OriginalPrice float64  `json:"original_price,omitempty"` // 原价
	Currency      string   `json:"currency,omitempty"`       // 币种
	Sales         int      `json:"sales,omitempty"`          // 销量
	Category      string   `json:"category,omitempty"`       // 分类
	Brand         string   `json:"brand,omitempty"`          // 品牌
	Tags          []string `json:"tags,omitempty"`           // 标签
	Rating        float64  `json:"rating,omitempty"`         // 评分
	ReviewCount   int      `json:"review_count,omitempty"`   // 评价数
}

// OrderInfo represents e-commerce order information
type OrderInfo struct {
	OrderID     string  `json:"order_id"`
	Status      string  `json:"status"` // pending, paid, shipped, delivered, cancelled
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency,omitempty"`
	ItemCount   int     `json:"item_count,omitempty"`
	CreatedAt   int64   `json:"created_at,omitempty"`
	PaidAt      int64   `json:"paid_at,omitempty"`
	ShippedAt   int64   `json:"shipped_at,omitempty"`
	DeliveredAt int64   `json:"delivered_at,omitempty"`
	UpdatedAt   int64   `json:"updated_at,omitempty"`
	Remark      string  `json:"remark,omitempty"`
}

// LogisticsInfo represents 发货/物流信息 including the 物流单号 and its trace
// timeline. It is exposed through the get_logistics tool.
type LogisticsInfo struct {
	OrderID     string           `json:"order_id"`
	TrackingNo  string           `json:"tracking_no"` // 物流单号
	Carrier     string           `json:"carrier,omitempty"`
	CarrierCode string           `json:"carrier_code,omitempty"`
	Status      string           `json:"status,omitempty"` // pending, in_transit, delivered, exception
	Traces      []LogisticsTrace `json:"traces,omitempty"`
	EstimatedAt int64            `json:"estimated_at,omitempty"`
	ShippedAt   int64            `json:"shipped_at,omitempty"`
	DeliveredAt int64            `json:"delivered_at,omitempty"`
	UpdatedAt   int64            `json:"updated_at,omitempty"`
}

// LogisticsTrace is a single node on the logistics timeline.
type LogisticsTrace struct {
	Time        int64  `json:"time"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
}

// ActivityContent represents 活动/事件信息 (promotion, flash sale, platform
// campaign). It is delivered through the activity.received event.
type ActivityContent struct {
	ActivityID string                 `json:"activity_id,omitempty"`
	EventType  string                 `json:"event_type,omitempty"` // promotion, flash_sale, campaign
	Title      string                 `json:"title,omitempty"`
	ItemID     string                 `json:"item_id,omitempty"`
	StartAt    int64                  `json:"start_at,omitempty"`
	EndAt      int64                  `json:"end_at,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

// Inquiry represents customer inquiry
type Inquiry struct {
	ProductID string `json:"product_id,omitempty"`
	OrderID   string `json:"order_id,omitempty"`
	Question  string `json:"question"`
	Category  string `json:"category,omitempty"` // price, shipping, quality, return, other
}

// LocationContent represents location message content
type LocationContent struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
	Name      string  `json:"name,omitempty"`
}

// EmojiContent represents emoji message content
type EmojiContent struct {
	Emoji string `json:"emoji"`
	Name  string `json:"name,omitempty"`
}

// MentionContent represents mention message content
type MentionContent struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// NewMessageChain creates a new message chain
func NewMessageChain(platform, instance string, sender SenderInfo) *MessageChain {
	mc := &MessageChain{
		ID:        GenerateEventID(),
		Timestamp: time.Now().UnixMilli(),
		Platform:  platform,
		Instance:  instance,
		Sender:    sender,
		Content:   make([]MessageElement, 0),
	}
	mc.Hash = mc.CalculateHash()
	return mc
}

// AddText adds a text element to the message chain
func (mc *MessageChain) AddText(text string) *MessageChain {
	mc.Content = append(mc.Content, MessageElement{
		Type:    ElementTypeText,
		Content: TextContent{Text: text},
	})
	mc.Hash = mc.CalculateHash()
	return mc
}

// AddImage adds an image element to the message chain
func (mc *MessageChain) AddImage(url string, width, height int) *MessageChain {
	mc.Content = append(mc.Content, MessageElement{
		Type: ElementTypeImage,
		Content: ImageContent{
			URL:    url,
			Width:  width,
			Height: height,
		},
	})
	mc.Hash = mc.CalculateHash()
	return mc
}

// AddProductCard adds a product card element to the message chain
func (mc *MessageChain) AddProductCard(card ProductCard) *MessageChain {
	mc.Content = append(mc.Content, MessageElement{
		Type:    ElementTypeProductCard,
		Content: card,
	})
	mc.Hash = mc.CalculateHash()
	return mc
}

// AddOrderInfo adds an order info element to the message chain
func (mc *MessageChain) AddOrderInfo(order OrderInfo) *MessageChain {
	mc.Content = append(mc.Content, MessageElement{
		Type:    ElementTypeOrderInfo,
		Content: order,
	})
	mc.Hash = mc.CalculateHash()
	return mc
}

// AddInquiry adds an inquiry element to the message chain
func (mc *MessageChain) AddInquiry(inquiry Inquiry) *MessageChain {
	mc.Content = append(mc.Content, MessageElement{
		Type:    ElementTypeInquiry,
		Content: inquiry,
	})
	mc.Hash = mc.CalculateHash()
	return mc
}

// AddLogistics adds a logistics element (发货/物流单号) to the message chain.
func (mc *MessageChain) AddLogistics(info LogisticsInfo) *MessageChain {
	mc.Content = append(mc.Content, MessageElement{
		Type:    ElementTypeLogistics,
		Content: info,
	})
	mc.Hash = mc.CalculateHash()
	return mc
}

// AddActivity adds an activity/event element to the message chain.
func (mc *MessageChain) AddActivity(activity ActivityContent) *MessageChain {
	mc.Content = append(mc.Content, MessageElement{
		Type:    ElementTypeActivity,
		Content: activity,
	})
	mc.Hash = mc.CalculateHash()
	return mc
}

// AddAudio adds an audio element (XianYuApis contentType=3) to the message chain.
func (mc *MessageChain) AddAudio(audio AudioContent) *MessageChain {
	if audio.URL == "" && audio.AudioURL != "" {
		audio.URL = audio.AudioURL
	}
	if audio.Duration == 0 && audio.DurationMS > 0 {
		audio.Duration = audio.DurationMS
	}
	mc.Content = append(mc.Content, MessageElement{
		Type:    ElementTypeAudio,
		Content: audio,
	})
	mc.Hash = mc.CalculateHash()
	return mc
}

// AddTradeCard adds a XianYu dxCard (contentType=26) trade card element.
func (mc *MessageChain) AddTradeCard(card TradeCard) *MessageChain {
	mc.Content = append(mc.Content, MessageElement{
		Type:    ElementTypeTradeCard,
		Content: card,
	})
	mc.Hash = mc.CalculateHash()
	return mc
}

// GetAudioContents returns all audio elements as AudioContent.
func (mc *MessageChain) GetAudioContents() []AudioContent {
	out := make([]AudioContent, 0)
	for _, elem := range mc.Content {
		if elem.Type != ElementTypeAudio {
			continue
		}
		switch content := elem.Content.(type) {
		case AudioContent:
			out = append(out, content)
		case map[string]interface{}:
			ac := AudioContent{}
			if v, ok := content["url"].(string); ok {
				ac.URL = v
			}
			if v, ok := content["audio_url"].(string); ok {
				ac.AudioURL = v
			}
			if v, ok := content["duration"].(float64); ok {
				ac.Duration = int(v)
			}
			if v, ok := content["duration_ms"].(float64); ok {
				ac.DurationMS = int(v)
			}
			if v, ok := content["mime_type"].(string); ok {
				ac.MimeType = v
			}
			out = append(out, ac)
		}
	}
	return out
}

// GetTradeCards returns all trade card elements as TradeCard.
func (mc *MessageChain) GetTradeCards() []TradeCard {
	out := make([]TradeCard, 0)
	for _, elem := range mc.Content {
		if elem.Type != ElementTypeTradeCard {
			continue
		}
		switch content := elem.Content.(type) {
		case TradeCard:
			out = append(out, content)
		case map[string]interface{}:
			tc := TradeCard{}
			if v, ok := content["target_url"].(string); ok {
				tc.TargetURL = v
			}
			if v, ok := content["title"].(string); ok {
				tc.Title = v
			}
			if v, ok := content["desc"].(string); ok {
				tc.Desc = v
			}
			if v, ok := content["detail_notice"].(string); ok {
				tc.DetailNotice = v
			}
			if v, ok := content["extension"].(map[string]interface{}); ok {
				tc.Extension = v
			}
			if v, ok := content["ext_json"].(map[string]interface{}); ok {
				tc.ExtJSON = v
			}
			if v, ok := content["item"].(map[string]interface{}); ok {
				item := &TradeCardItem{}
				if s, ok := v["click_param"].(string); ok {
					item.ClickParam = s
				}
				if s, ok := v["target_url"].(string); ok {
					item.TargetURL = s
				}
				if s, ok := v["ex_content"].(map[string]interface{}); ok {
					item.ExContent = s
				}
				tc.Item = item
			}
			out = append(out, tc)
		}
	}
	return out
}

// CalculateHash calculates the MD5 hash of the message chain
func (mc *MessageChain) CalculateHash() string {
	data, _ := json.Marshal(struct {
		ID        string           `json:"id"`
		Timestamp int64            `json:"timestamp"`
		Platform  string           `json:"platform"`
		Instance  string           `json:"instance"`
		Sender    SenderInfo       `json:"sender"`
		Content   []MessageElement `json:"content"`
	}{
		ID:        mc.ID,
		Timestamp: mc.Timestamp,
		Platform:  mc.Platform,
		Instance:  mc.Instance,
		Sender:    mc.Sender,
		Content:   mc.Content,
	})

	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// VerifyHash verifies the message chain hash
func (mc *MessageChain) VerifyHash() bool {
	expectedHash := mc.CalculateHash()
	return mc.Hash == expectedHash
}

// GetTextContent returns the first text content. It handles both the typed
// TextContent struct (when the chain is built in-process) and the generic
// map[string]interface{} form (when the chain is parsed from JSON).
func (mc *MessageChain) GetTextContent() string {
	for _, elem := range mc.Content {
		if elem.Type != ElementTypeText {
			continue
		}
		switch content := elem.Content.(type) {
		case TextContent:
			return content.Text
		case map[string]interface{}:
			if text, ok := content["text"].(string); ok {
				return text
			}
		}
	}
	return ""
}

// GetProductCards returns all product cards. It handles both the typed
// ProductCard struct and the generic map form parsed from JSON.
func (mc *MessageChain) GetProductCards() []ProductCard {
	cards := make([]ProductCard, 0)
	for _, elem := range mc.Content {
		if elem.Type != ElementTypeProductCard {
			continue
		}
		switch content := elem.Content.(type) {
		case ProductCard:
			cards = append(cards, content)
		case map[string]interface{}:
			cards = append(cards, productCardFromMap(content))
		}
	}
	return cards
}

// productCardFromMap converts a generic map into a ProductCard.
func productCardFromMap(m map[string]interface{}) ProductCard {
	card := ProductCard{}
	if v, ok := m["item_id"].(string); ok {
		card.ItemID = v
	}
	if v, ok := m["title"].(string); ok {
		card.Title = v
	}
	if v, ok := m["price"].(float64); ok {
		card.Price = v
	}
	if v, ok := m["image_url"].(string); ok {
		card.ImageURL = v
	}
	if v, ok := m["detail_url"].(string); ok {
		card.DetailURL = v
	}
	if v, ok := m["platform"].(string); ok {
		card.Platform = v
	}
	if v, ok := m["sku"].(string); ok {
		card.SKU = v
	}
	if v, ok := m["stock"].(float64); ok {
		card.Stock = int(v)
	}
	// 非活动字段（可主动查询）。
	if v, ok := m["description"].(string); ok {
		card.Description = v
	}
	if v, ok := m["images"].([]interface{}); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				card.Images = append(card.Images, s)
			}
		}
	}
	if v, ok := m["original_price"].(float64); ok {
		card.OriginalPrice = v
	}
	if v, ok := m["currency"].(string); ok {
		card.Currency = v
	}
	if v, ok := m["sales"].(float64); ok {
		card.Sales = int(v)
	}
	if v, ok := m["category"].(string); ok {
		card.Category = v
	}
	if v, ok := m["brand"].(string); ok {
		card.Brand = v
	}
	if v, ok := m["tags"].([]interface{}); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				card.Tags = append(card.Tags, s)
			}
		}
	}
	if v, ok := m["rating"].(float64); ok {
		card.Rating = v
	}
	if v, ok := m["review_count"].(float64); ok {
		card.ReviewCount = int(v)
	}
	return card
}

// ToJSON converts the message chain to JSON
func (mc *MessageChain) ToJSON() ([]byte, error) {
	return json.Marshal(mc)
}

// MessageChainFromJSON parses JSON into message chain
func MessageChainFromJSON(data []byte) (*MessageChain, error) {
	var mc MessageChain
	if err := json.Unmarshal(data, &mc); err != nil {
		return nil, err
	}
	return &mc, nil
}

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
	ElementTypeInquiry     ElementType = "inquiry"
	ElementTypeLocation    ElementType = "location"
	ElementTypeEmoji       ElementType = "emoji"
	ElementTypeMention     ElementType = "mention"
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

// AudioContent represents audio message content
type AudioContent struct {
	URL      string `json:"url"`
	Duration int    `json:"duration,omitempty"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
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

// ProductCard represents e-commerce product card
type ProductCard struct {
	ItemID    string  `json:"item_id"`
	Title     string  `json:"title"`
	Price     float64 `json:"price"`
	ImageURL  string  `json:"image_url"`
	DetailURL string  `json:"detail_url"`
	Platform  string  `json:"platform,omitempty"`
	SKU       string  `json:"sku,omitempty"`
	Stock     int     `json:"stock,omitempty"`
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

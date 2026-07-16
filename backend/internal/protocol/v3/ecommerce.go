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

// CreateProductCard creates a product card from product detail
func CreateProductCard(detail ProductDetail) ProductCard {
	return ProductCard{
		ItemID:    detail.ItemID,
		Title:     detail.Title,
		Price:     detail.Price,
		ImageURL:  detail.ImageURL,
		DetailURL: detail.DetailURL,
		Platform:  detail.Platform,
		SKU:       detail.SKU,
		Stock:     detail.Stock,
	}
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

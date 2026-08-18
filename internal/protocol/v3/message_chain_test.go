package v3

import (
	"encoding/json"
	"testing"
)

// TestGetTextContentTyped verifies text extraction from an in-process chain
// (typed TextContent structs).
func TestGetTextContentTyped(t *testing.T) {
	mc := NewMessageChain("xianyu", "inst-1", SenderInfo{ID: "u1", Name: "买家"})
	mc.AddText("你好")
	mc.AddImage("https://example.com/a.jpg", 100, 100)

	if got := mc.GetTextContent(); got != "你好" {
		t.Fatalf("expected '你好', got %q", got)
	}
}

// TestGetTextContentFromJSON verifies text extraction from a chain parsed from
// JSON, where element Content is a generic map[string]interface{}.
func TestGetTextContentFromJSON(t *testing.T) {
	raw := `{
		"id": "chain-1",
		"timestamp": 1723891200000,
		"platform": "xianyu",
		"instance": "inst-1",
		"sender": {"id": "u1", "name": "买家"},
		"content": [
			{"type": "text", "content": {"text": "这个商品有货吗？"}},
			{"type": "image", "content": {"url": "https://example.com/a.jpg"}}
		]
	}`
	mc, err := MessageChainFromJSON([]byte(raw))
	if err != nil {
		t.Fatalf("failed to parse chain: %v", err)
	}
	if got := mc.GetTextContent(); got != "这个商品有货吗？" {
		t.Fatalf("expected '这个商品有货吗？', got %q", got)
	}
}

// TestGetProductCardsFromJSON verifies product card extraction from a chain
// parsed from JSON.
func TestGetProductCardsFromJSON(t *testing.T) {
	raw := `{
		"id": "chain-2",
		"timestamp": 1723891200000,
		"platform": "taobao",
		"instance": "inst-2",
		"sender": {"id": "u2", "name": "卖家"},
		"content": [
			{"type": "product_card", "content": {
				"item_id": "123456",
				"title": "iPhone 15 Pro",
				"price": 7999.0,
				"image_url": "https://example.com/iphone.jpg",
				"detail_url": "https://example.com/item/123456",
				"platform": "taobao",
				"sku": "SKU-001",
				"stock": 100
			}}
		]
	}`
	mc, err := MessageChainFromJSON([]byte(raw))
	if err != nil {
		t.Fatalf("failed to parse chain: %v", err)
	}
	cards := mc.GetProductCards()
	if len(cards) != 1 {
		t.Fatalf("expected 1 product card, got %d", len(cards))
	}
	card := cards[0]
	if card.ItemID != "123456" || card.Title != "iPhone 15 Pro" {
		t.Fatalf("unexpected card: %+v", card)
	}
	if card.Price != 7999.0 {
		t.Fatalf("expected price 7999.0, got %v", card.Price)
	}
	if card.Stock != 100 {
		t.Fatalf("expected stock 100, got %d", card.Stock)
	}
}

// TestParseInboundMessagePayload verifies that parsing a full V3 envelope
// payload derives message type and text content from the message chain.
func TestParseInboundMessagePayload(t *testing.T) {
	raw := `{
		"id": "msg-1",
		"timestamp": 1723891200000,
		"platform": "xianyu",
		"instance": "inst-1",
		"conversation_id": "conv-1",
		"sender": {"id": "u1", "name": "买家"},
		"message_chain": {
			"id": "chain-1",
			"timestamp": 1723891200000,
			"platform": "xianyu",
			"instance": "inst-1",
			"sender": {"id": "u1", "name": "买家"},
			"content": [
				{"type": "text", "content": {"text": "你好"}}
			]
		}
	}`
	p, err := ParseInboundMessagePayload([]byte(raw))
	if err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if p.MessageType != "text" {
		t.Fatalf("expected message_type 'text', got %q", p.MessageType)
	}
	if p.MessageContent != "你好" {
		t.Fatalf("expected message_content '你好', got %q", p.MessageContent)
	}
}

// TestInboundPayloadToMap verifies the payload can be serialized back to a map
// without losing the message chain.
func TestInboundPayloadToMap(t *testing.T) {
	p := NewInboundMessagePayload("xianyu", "inst-1", "conv-1", SenderInfo{ID: "u1", Name: "买家"})
	mc := NewMessageChain("xianyu", "inst-1", SenderInfo{ID: "u1", Name: "买家"})
	mc.AddText("你好")
	p.SetMessageChain(mc)

	m, err := p.ToMap()
	if err != nil {
		t.Fatalf("failed to convert to map: %v", err)
	}
	if m["platform"] != "xianyu" {
		t.Fatalf("expected platform 'xianyu', got %v", m["platform"])
	}
	if _, ok := m["message_chain"]; !ok {
		t.Fatal("expected message_chain to be preserved in map")
	}

	// Round-trip through JSON to ensure the chain survives serialization.
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal map: %v", err)
	}
	parsed, err := ParseInboundMessagePayload(data)
	if err != nil {
		t.Fatalf("failed to re-parse: %v", err)
	}
	if parsed.MessageContent != "你好" {
		t.Fatalf("expected re-parsed content '你好', got %q", parsed.MessageContent)
	}
}

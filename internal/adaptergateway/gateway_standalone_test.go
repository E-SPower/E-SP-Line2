package adaptergateway

import (
	"testing"

	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// TestRouteOutboundToStandalone pins the outbound path for standalone adapters:
// they never register a bridge WebSocket, so without the hook every reply would
// be dropped with "no bridge connection".
func TestRouteOutboundToStandalone(t *testing.T) {
	logger.Init("error")
	g := NewGateway(DefaultGatewayConfig(), nil)

	var got map[string]interface{}
	g.SetStandaloneSender(func(instanceID string, command map[string]interface{}) bool {
		if instanceID != "inst-1" {
			t.Fatalf("instance id = %q", instanceID)
		}
		got = command
		return true
	})

	frame := map[string]interface{}{
		"payload": map[string]interface{}{
			"instance_id":     "inst-1",
			"command_type":    "send_text",
			"conversation_id": "c1",
			"sender_id":       "u1",
			"message_chain":   []interface{}{map[string]interface{}{"type": "text", "content": map[string]interface{}{"text": "hi"}}},
		},
	}
	if !g.RouteOutboundToBridge("inst-1", frame) {
		t.Fatal("command was not delivered to the standalone runtime")
	}
	if got == nil {
		t.Fatal("hook received no command")
	}
	if got["command_type"] != "send_text" {
		t.Fatalf("command_type = %v", got["command_type"])
	}
	payload, _ := got["payload"].(map[string]interface{})
	if payload == nil || payload["text"] != "hi" {
		t.Fatalf("text not carried to the adapter: %#v", got)
	}
}

// TestRouteOutboundUnknownInstanceStillWarns: no bridge and no hook must stay a
// false return, not a panic.
func TestRouteOutboundUnknownInstance(t *testing.T) {
	logger.Init("error")
	g := NewGateway(DefaultGatewayConfig(), nil)
	if g.RouteOutboundToBridge("nope", map[string]interface{}{}) {
		t.Fatal("unknown instance must not report delivery")
	}
	g.SetStandaloneSender(func(string, map[string]interface{}) bool { return false })
	if g.RouteOutboundToBridge("nope", map[string]interface{}{}) {
		t.Fatal("hook said no, must report not delivered")
	}
}

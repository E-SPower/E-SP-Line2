package message

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
)

// SendCommand must report an undelivered command as an error, otherwise the
// caller records a command nobody will ever send as "sent".
func TestSendCommandFailsWithoutConnection(t *testing.T) {
	hub := NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	err := hub.SendCommand("instance-absent", v3.NewOutboundCommand("send_text", "instance-absent", "conv", nil))
	if err == nil {
		t.Fatal("SendCommand reported success with no adapter connection")
	}
	if !strings.Contains(err.Error(), "instance-absent") {
		t.Fatalf("error does not name the instance: %v", err)
	}
}

// With a matching adapter connection the command envelope is written to it.
func TestSendCommandReachesAdapterConnection(t *testing.T) {
	hub := NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	conn := &Connection{ID: "ad-1", Type: "adapter", InstanceID: "instance-1", Send: make(chan []byte, 4)}
	hub.GetDispatcher().RegisterConnection(conn)

	if err := hub.SendCommand("instance-1", v3.NewOutboundCommand("send_text", "instance-1", "conv-1", nil)); err != nil {
		t.Fatalf("SendCommand: %v", err)
	}

	select {
	case frame := <-conn.Send:
		var envelope v3.MessageEnvelope
		if err := json.Unmarshal(frame, &envelope); err != nil {
			t.Fatalf("frame is not an envelope: %v", err)
		}
		if envelope.EventType != v3.EventCommandCreated {
			t.Fatalf("event_type = %q, want %q", envelope.EventType, v3.EventCommandCreated)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("command was not written to the connection")
	}
}

// A command addressed at another instance must not leak into this connection.
func TestSendCommandDoesNotCrossInstances(t *testing.T) {
	hub := NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	conn := &Connection{ID: "ad-2", Type: "adapter", InstanceID: "instance-2", Send: make(chan []byte, 4)}
	hub.GetDispatcher().RegisterConnection(conn)

	if err := hub.SendCommand("instance-other", v3.NewOutboundCommand("send_text", "instance-other", "c", nil)); err == nil {
		t.Fatal("SendCommand matched the wrong instance")
	}
	select {
	case frame := <-conn.Send:
		t.Fatalf("unexpected frame on instance-2: %s", frame)
	default:
	}
}

// Routed messages reach the app socket selected by the rule.
func TestProcessMessageDispatchesToRoutedApp(t *testing.T) {
	hub := NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	hub.SyncRules([]*RouteRule{{ID: "r1", TargetType: "app", TargetID: "app-1", Enabled: true}})

	conn := &Connection{ID: "app-1", Type: "app", AppID: "app-1", Send: make(chan []byte, 4)}
	hub.GetDispatcher().RegisterConnection(conn)

	envelope := v3.NewMessageEnvelope("platform-xianyu", "instance-1", v3.EventMessageReceived,
		map[string]interface{}{"message_content": "hi"})
	if err := hub.ProcessMessage(envelope); err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	select {
	case <-conn.Send:
	case <-time.After(2 * time.Second):
		t.Fatal("message was not delivered to the routed app")
	}
}

// A disabled rule must not match.
func TestProcessMessageSkipsDisabledRule(t *testing.T) {
	hub := NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	hub.SyncRules([]*RouteRule{{ID: "r1", TargetType: "app", TargetID: "app-1", Enabled: false}})
	conn := &Connection{ID: "app-1", Type: "app", AppID: "app-1", Send: make(chan []byte, 4)}
	hub.GetDispatcher().RegisterConnection(conn)

	envelope := v3.NewMessageEnvelope("platform-xianyu", "instance-1", v3.EventMessageReceived, nil)
	if err := hub.ProcessMessage(envelope); err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	select {
	case frame := <-conn.Send:
		t.Fatalf("disabled rule delivered: %s", frame)
	default:
	}
}

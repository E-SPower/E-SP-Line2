package messagefields

import (
	"strings"
	"testing"
)

func TestEveryFieldIsDocumented(t *testing.T) {
	for kind, o := range Objects() {
		for _, f := range o.Fields {
			if f.Summary == "" {
				t.Fatalf("%s.%s misses a summary", kind, f.Path)
			}
			if f.Type == "" {
				t.Fatalf("%s.%s misses a type", kind, f.Path)
			}
			if f.Nullable && f.Required {
				// 合法：字段总是存在，取不到时是 null。
				continue
			}
			if !f.Required {
				t.Fatalf("%s.%s is neither required nor nullable", kind, f.Path)
			}
		}
	}
}

func TestEveryObjectHasKind(t *testing.T) {
	for key, o := range Objects() {
		if o.Kind == "" {
			t.Fatalf("object %q misses a kind", key)
		}
		if key != o.Kind {
			t.Fatalf("catalog key %q != object kind %q", key, o.Kind)
		}
	}
}

// 消息对象必须带全部正文元素的可填充字段。
func TestMessageHasBodyFields(t *testing.T) {
	msg, ok := ObjectOf(KindMessage)
	if !ok {
		t.Fatal("message object missing")
	}
	for _, want := range []string{"from", "type", "text", "media", "created_at"} {
		f, ok := msg.Field(want)
		if !ok {
			t.Fatalf("message misses %q", want)
		}
		if !f.MessageField {
			t.Fatalf("message.%s is not marked as a body field", want)
		}
	}
	if len(msg.MessageFields()) != 5 {
		t.Fatalf("message has %d body fields, want 5", len(msg.MessageFields()))
	}
}

func TestEnumsAreNonEmpty(t *testing.T) {
	for kind, o := range Objects() {
		for _, f := range o.Fields {
			if f.Type == "" {
				continue
			}
			if strings.HasSuffix(f.Path, "type") && len(f.Values) == 0 {
				t.Fatalf("%s.%s looks like an enum but has no values", kind, f.Path)
			}
		}
	}
}

func TestElements(t *testing.T) {
	for _, want := range []string{ElementText, ElementImage, ElementProductCard, ElementOrderInfo} {
		if _, ok := Elements[want]; !ok {
			t.Fatalf("element %q missing", want)
		}
	}
}

func TestAllFields(t *testing.T) {
	all := AllFields()
	if len(all) == 0 {
		t.Fatal("AllFields is empty")
	}
	if _, ok := all["message.text"]; !ok {
		t.Fatal("AllFields misses message.text")
	}
}

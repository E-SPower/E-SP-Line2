package messagefields

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// docPath 是字段文档在当前测试工作目录下的位置。
const docPath = "../../docs/developer-guide/message-fields.md"

// 每个字段路径都要在文档里出现，否则等于没写。
func TestDocsCoverEveryField(t *testing.T) {
	raw, err := os.ReadFile(filepath.Clean(docPath))
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	doc := string(raw)

	for kind, o := range Objects() {
		for _, f := range o.Fields {
			// 文档用反引号包字段名来避免歧义：message.text 写成 `text`，
			// 嵌套字段用完整路径 `stats.views`。
			if !strings.Contains(doc, "`"+f.Path+"`") && !strings.Contains(doc, "`"+kind+"."+f.Path+"`") {
				t.Errorf("%s.%s is not documented as `%s`", kind, f.Path, f.Path)
			}
		}
	}
}

// 文档的枚举取值要和代码一致，增删必须两边一起改。
func TestDocsMatchEnums(t *testing.T) {
	raw, err := os.ReadFile(filepath.Clean(docPath))
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	doc := string(raw)

	for kind, o := range Objects() {
		for _, f := range o.Fields {
			for _, v := range f.Values {
				if !strings.Contains(doc, "`"+v+"`") {
					t.Errorf("enum value %q of %s.%s is not documented", v, kind, f.Path)
				}
			}
		}
	}
}

// 每个对象都要在文档里有一节。
func TestDocsCoverEveryObject(t *testing.T) {
	raw, err := os.ReadFile(filepath.Clean(docPath))
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	doc := string(raw)

	for kind, o := range Objects() {
		if !strings.Contains(doc, o.Kind) && !strings.Contains(doc, "`"+kind+"`") {
			t.Errorf("object %q has no section", kind)
		}
	}
}

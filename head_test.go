package inertia

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmptyServerHeadSerializesAsArray(t *testing.T) {
	body, err := json.Marshal(ServerHead{})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "[]" {
		t.Fatalf("empty server head must be an array: %s", body)
	}
}

func TestServerHeadElementsReturnsCopy(t *testing.T) {
	head, err := NewServerHead(HeadTitle("Users"))
	if err != nil {
		t.Fatal(err)
	}
	elements := head.Elements()
	elements[0] = "changed"
	if head.Elements()[0] == "changed" {
		t.Fatal("server head elements must not expose mutable internal state")
	}
}

func TestServerHeadBuilderEscapesContentAndAttributes(t *testing.T) {
	head, err := NewServerHead(
		HeadTitle(`<Admin & Users>`),
		HeadMeta("description", `Users "quoted" <unsafe>`),
		HeadMetaProperty("og:title", `Admin & Users`),
		HeadLink("canonical", `https://example.com/users?a=1&b=2`),
		HeadScript(`https://example.com/app.js?a=1&b=2`).Attr("nonce", `nonce"value`),
	)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(head.Elements(), "\n")
	for _, expected := range []string{
		`<title data-inertia="title">&lt;Admin &amp; Users&gt;</title>`,
		`content="Users &#34;quoted&#34; &lt;unsafe&gt;"`,
		`property="og:title"`,
		`href="https://example.com/users?a=1&amp;b=2"`,
		`nonce="nonce&#34;value"`,
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing escaped head fragment %q: %s", expected, joined)
		}
	}
	if strings.Contains(joined, `<unsafe>`) {
		t.Fatalf("unescaped head content: %s", joined)
	}
}

func TestServerHeadBuilderRejectsInvalidAttributeNames(t *testing.T) {
	_, err := NewServerHead(HeadScript("/app.js").Attr(`src onerror`, "alert(1)"))
	if !errors.Is(err, ErrInvalidHeadElement) {
		t.Fatalf("expected ErrInvalidHeadElement, got %v", err)
	}
}

func TestRawHeadElementRequiresExplicitTrustedHTML(t *testing.T) {
	head, err := NewServerHead(RawHeadElement(template.HTML(`<meta name="custom" content="trusted">`)))
	if err != nil {
		t.Fatal(err)
	}
	if elements := head.Elements(); len(elements) != 1 || elements[0] != `<meta data-inertia="server-head-0" name="custom" content="trusted">` {
		t.Fatalf("unexpected raw head: %#v", head)
	}
}

func TestRawHeadElementOnlyDetectsOpeningTagAttributes(t *testing.T) {
	for name, raw := range map[string]template.HTML{
		"attribute value": template.HTML(`<meta content="mentions data-inertia=value">`),
		"element content": template.HTML(`<title>mentions data-inertia=value</title>`),
	} {
		t.Run(name, func(t *testing.T) {
			head, err := NewServerHead(RawHeadElement(raw))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(head.Elements()[0], ` data-inertia="server-head-0"`) {
				t.Fatalf("missing generated key: %s", head.Elements()[0])
			}
		})
	}

	head, err := NewServerHead(RawHeadElement(template.HTML(`<meta data-inertia='custom-key' content="trusted">`)))
	if err != nil {
		t.Fatal(err)
	}
	if head.Elements()[0] != `<meta data-inertia='custom-key' content="trusted">` {
		t.Fatalf("existing opening-tag key should be preserved: %s", head.Elements()[0])
	}
}

func TestWithServerHeadUsesConfiguredPropAndPartialReload(t *testing.T) {
	head, err := NewServerHead(HeadTitle("Users"))
	if err != nil {
		t.Fatal(err)
	}
	renderer := newTestRenderer(t, Config{ServerHeadProp: "metaTags"})
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Users/Index")
	req.Header.Set(HeaderInertiaPartialData, "users")
	w := httptest.NewRecorder()

	err = renderer.Render(w, req, "Users/Index", Props{"users": []string{"hiro"}}, WithServerHead(head))
	if err != nil {
		t.Fatal(err)
	}
	page := decodePage(t, w)
	values, ok := page.Props["metaTags"].([]any)
	if !ok || len(values) != 1 || values[0] != `<title data-inertia="title">Users</title>` {
		t.Fatalf("unexpected server head prop: %#v", page.Props["metaTags"])
	}
}

func TestServerHeadNamedPropsValidatesName(t *testing.T) {
	head, buildErr := NewServerHead(HeadTitle("Users"))
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	_, err := head.NamedProps("meta.tags")
	if !errors.Is(err, ErrInvalidServerHeadProp) {
		t.Fatalf("expected ErrInvalidServerHeadProp, got %v", err)
	}
}

func TestNewRejectsInvalidServerHeadProp(t *testing.T) {
	_, err := New(Config{
		RootView:       rootViewFunc(func(w io.Writer, data RootViewData) error { return nil }),
		ServerHeadProp: "meta.tags",
	})
	if !errors.Is(err, ErrInvalidServerHeadProp) {
		t.Fatalf("expected ErrInvalidServerHeadProp, got %v", err)
	}
}

func TestServerHeadBuilderRejectsExecutableAttributesAndMissingValues(t *testing.T) {
	for name, element := range map[string]HeadElement{
		"event attribute":   HeadLink("canonical", "/users").Attr("onload", "alert(1)"),
		"empty key":         HeadTitle("Users").Key(""),
		"empty meta name":   HeadMeta("", "value"),
		"empty link href":   HeadLink("canonical", ""),
		"empty script src":  HeadScript(""),
		"empty raw element": RawHeadElement(template.HTML("")),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewServerHead(element)
			if !errors.Is(err, ErrInvalidHeadElement) {
				t.Fatalf("expected ErrInvalidHeadElement, got %v", err)
			}
		})
	}
}

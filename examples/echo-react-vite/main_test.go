package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	echo "github.com/labstack/echo/v5"
	inertia "github.com/mayahiro/go-inertia"
	inertiaecho "github.com/mayahiro/go-inertia/adapters/echo"
)

func TestBindCreateUserAcceptsJSON(t *testing.T) {
	input, err := bindCreateUserContext(`{"name":"Ada Lovelace","email":"ada@example.com"}`, echo.MIMEApplicationJSON)
	if err != nil {
		t.Fatal(err)
	}

	if input.Name != "Ada Lovelace" || input.Email != "ada@example.com" {
		t.Fatalf("unexpected input: %#v", input)
	}
}

func TestBindCreateUserAcceptsForm(t *testing.T) {
	input, err := bindCreateUserContext("name=Grace+Hopper&email=grace%40example.com", echo.MIMEApplicationForm)
	if err != nil {
		t.Fatal(err)
	}

	if input.Name != "Grace Hopper" || input.Email != "grace@example.com" {
		t.Fatalf("unexpected input: %#v", input)
	}
}

func TestCreatedUserAppearsOnFirstPage(t *testing.T) {
	users := prependUser(seedUsers(), createUserInput{
		Name:  "New User",
		Email: "new@example.com",
	})

	page := paginateUsers(httptest.NewRequest(http.MethodGet, "/users", nil), users)

	if len(page.Data) == 0 {
		t.Fatal("expected first page users")
	}
	if page.Data[0].Name != "New User" || page.Data[0].Email != "new@example.com" {
		t.Fatalf("created user should be first: %#v", page.Data[0])
	}
}

func TestPageHeadEscapesDynamicValues(t *testing.T) {
	head, err := pageHead("<Users>", `Manage "users"`)
	if err != nil {
		t.Fatal(err)
	}

	elements := head.Elements()
	if len(elements) != 2 {
		t.Fatalf("unexpected head elements: %#v", elements)
	}
	joined := strings.Join(elements, "")
	if !strings.Contains(joined, "&lt;Users&gt;") || !strings.Contains(joined, "&#34;users&#34;") {
		t.Fatalf("head values were not escaped: %s", joined)
	}
	if !strings.Contains(joined, "data-inertia") {
		t.Fatalf("head elements need stable keys: %s", joined)
	}
}

func TestLoadServerTimeCanDemonstrateRescue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?failServerTime=1", nil)

	if _, err := loadServerTime(req); err == nil {
		t.Fatal("expected demonstration deferred prop failure")
	}
}

func TestDevToolsEnabled(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "true", want: true},
		{value: " TRUE ", want: true},
		{value: "1", want: true},
		{value: "false", want: false},
		{value: "", want: false},
		{value: "invalid", want: false},
	}

	for _, test := range tests {
		if got := devToolsEnabled(test.value); got != test.want {
			t.Fatalf("devToolsEnabled(%q) = %t, want %t", test.value, got, test.want)
		}
	}
}

func TestEchoDevToolsMetadata(t *testing.T) {
	view := template.Must(template.New("app").Parse(`<!doctype html><html><body>{{ .InertiaApp }}</body></html>`))
	renderer, err := inertia.New(inertia.Config{
		RootView: inertia.NewTemplateRootView(view, "app"),
		DevTools: inertia.DevToolsConfig{
			Enabled: true,
			ComponentPathResolver: func(component string) string {
				return "resources/js/Pages/" + component + ".tsx"
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	app := inertiaecho.New(renderer)
	e := echo.New()
	e.Use(app.Middleware)
	e.GET("/users/:id", func(c *echo.Context) error {
		return app.Render(c, "Users/Show", inertia.Props{"id": c.Param("id")})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	if got := w.Code; got != http.StatusOK {
		t.Fatalf("unexpected application status %d: %s", got, w.Body.String())
	}
	id := w.Header().Get(inertia.HeaderInertiaDevToolsID)
	if id == "" {
		t.Fatal("missing DevTools entry id")
	}

	entryRequest := httptest.NewRequest(http.MethodGet, "/_inertia/devtools/entries/"+id, nil)
	entryRequest.RemoteAddr = "127.0.0.1:12345"
	entryResponse := httptest.NewRecorder()
	e.ServeHTTP(entryResponse, entryRequest)
	if got := entryResponse.Code; got != http.StatusOK {
		t.Fatalf("unexpected entry status %d: %s", got, entryResponse.Body.String())
	}
	var entry struct {
		Meta struct {
			Component *string `json:"component"`
		} `json:"__meta"`
		Route struct {
			Name *string `json:"name"`
			URI  string  `json:"uri"`
		} `json:"route"`
		RenderSource *struct {
			File string `json:"file"`
			Line int    `json:"line"`
		} `json:"renderSource"`
		ComponentPath *string `json:"componentPath"`
	}
	if err := json.Unmarshal(entryResponse.Body.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Meta.Component == nil || *entry.Meta.Component != "Users/Show" {
		t.Fatalf("unexpected component %#v", entry.Meta.Component)
	}
	if entry.Route.Name != nil || entry.Route.URI != "/users/:id" {
		t.Fatalf("unexpected Echo route metadata %#v", entry.Route)
	}
	if entry.RenderSource == nil || !strings.HasSuffix(entry.RenderSource.File, "main_test.go") || entry.RenderSource.Line < 1 {
		t.Fatalf("unexpected render source %#v", entry.RenderSource)
	}
	if entry.ComponentPath == nil || *entry.ComponentPath != "resources/js/Pages/Users/Show.tsx" {
		t.Fatalf("unexpected component path %#v", entry.ComponentPath)
	}
}

func bindCreateUserContext(body string, contentType string) (createUserInput, error) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return bindCreateUser(c)
}

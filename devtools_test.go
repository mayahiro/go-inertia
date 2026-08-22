package inertia

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDevToolsRecordsInitialInertiaResponse(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{
		SharedProps: StaticSharedProps(Props{
			"account": Always(map[string]any{
				"name":  "Ada",
				"token": "shared-secret",
			}),
		}),
		DevTools: DevToolsConfig{
			Enabled: true,
			ComponentPathResolver: func(component string) string {
				return "resources/js/Pages/" + component + ".tsx"
			},
		},
	})
	req := newLocalDevToolsRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set("Authorization", "Bearer request-secret")
	w := httptest.NewRecorder()
	var renderErr error

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Set-Cookie", "session=response-secret")
		renderer.RecordDevToolsRoute(req, "dashboard", "/dashboard", "")
		renderer.RecordDevToolsRenderSource(req, "handlers/dashboard.go", 42)
		renderErr = renderer.Render(w, req, "Dashboard", Props{
			"profile": map[string]any{
				"email":    "ada@example.com",
				"password": "prop-secret",
			},
			"stats": Computed(func(*http.Request) (any, error) {
				return 42, nil
			}).Defer("analytics"),
		})
	})).ServeHTTP(w, req)
	if renderErr != nil {
		t.Fatal(renderErr)
	}

	id := w.Header().Get(HeaderInertiaDevToolsID)
	if len(id) != 32 {
		t.Fatalf("unexpected entry id %q", id)
	}
	if got := w.Header().Get(HeaderInertiaDevToolsParentOut); got != id {
		t.Fatalf("unexpected parent-out %q", got)
	}
	if got := w.Code; got != http.StatusOK {
		t.Fatalf("unexpected response status %d", got)
	}
	expectedScript := `<script data-inertia-devtools-id type="application/json">"` + id + `"</script>`
	if !strings.Contains(w.Body.String(), expectedScript) {
		t.Fatalf("initial response is missing discovery script: %s", w.Body.String())
	}

	entry := readDevToolsEntry(t, renderer, id)
	if entry.Meta.ID != id || entry.Meta.RequestType != "initial" {
		t.Fatalf("unexpected entry metadata %#v", entry.Meta)
	}
	if entry.Meta.Component == nil || *entry.Meta.Component != "Dashboard" {
		t.Fatalf("unexpected component %#v", entry.Meta.Component)
	}
	if entry.Meta.Status != http.StatusOK || entry.Meta.Method != http.MethodGet {
		t.Fatalf("unexpected HTTP metadata %#v", entry.Meta)
	}
	if entry.Meta.URL != "http://example.com/dashboard" {
		t.Fatalf("unexpected URL %q", entry.Meta.URL)
	}
	if entry.ComponentPath == nil || *entry.ComponentPath != "resources/js/Pages/Dashboard.tsx" {
		t.Fatalf("unexpected component path %#v", entry.ComponentPath)
	}
	if entry.Route.Name == nil || *entry.Route.Name != "dashboard" || entry.Route.URI != "/dashboard" {
		t.Fatalf("unexpected route %#v", entry.Route)
	}
	if entry.RenderSource == nil || entry.RenderSource.File != "handlers/dashboard.go" || entry.RenderSource.Line != 42 {
		t.Fatalf("unexpected render source %#v", entry.RenderSource)
	}
	if got := entry.HTTP.RequestHeaders["authorization"]; got != devToolsRedactedValue {
		t.Fatalf("authorization header was not redacted: %q", got)
	}
	if got := entry.HTTP.ResponseHeaders["set-cookie"]; got != devToolsRedactedValue {
		t.Fatalf("set-cookie header was not redacted: %q", got)
	}
	if got := entry.HTTP.ResponseHeaders[strings.ToLower(HeaderInertiaDevToolsID)]; got != id {
		t.Fatalf("response headers are missing entry id: %q", got)
	}
	if entry.HTTP.RequestBody.Status != "empty" || entry.HTTP.ResponseBody.Status != "present" {
		t.Fatalf("unexpected body capture %#v", entry.HTTP)
	}
	page := devToolsBodyMap(t, entry.HTTP.ResponseBody)
	pageProps, ok := page["props"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected page props %#v", page["props"])
	}
	profile, ok := pageProps["profile"].(map[string]any)
	if !ok || profile["password"] != devToolsRedactedValue {
		t.Fatalf("profile secret was not redacted: %#v", pageProps["profile"])
	}
	account, ok := pageProps["account"].(map[string]any)
	if !ok || account["token"] != devToolsRedactedValue {
		t.Fatalf("shared secret was not redacted: %#v", pageProps["account"])
	}
	if got := entry.Props["account"]; got.InertiaType != "always" || !got.Shared {
		t.Fatalf("unexpected shared prop metadata %#v", got)
	}
	if got := entry.Props["stats"]; got.InertiaType != "defer" || got.DeferGroup != "analytics" {
		t.Fatalf("unexpected deferred prop metadata %#v", got)
	}
	if got := entry.Props["errors"]; got.InertiaType != "always" || !got.Shared {
		t.Fatalf("unexpected error prop metadata %#v", got)
	}
}

func TestDevToolsRecordsDeferredRequestCorrelation(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	req := newLocalDevToolsRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaDevToolsTab, "tab-1")
	req.Header.Set(HeaderInertiaDevToolsVisit, "visit-1")
	req.Header.Set(HeaderInertiaDevToolsParent, "root-entry")
	req.Header.Set(HeaderInertiaDevToolsDeferred, "analytics")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "stats")
	w := httptest.NewRecorder()
	var renderErr error

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		renderErr = renderer.Render(w, req, "Dashboard", Props{
			"stats": Computed(func(*http.Request) (any, error) {
				return 42, nil
			}).Defer("analytics"),
		})
	})).ServeHTTP(w, req)
	if renderErr != nil {
		t.Fatal(renderErr)
	}

	id := w.Header().Get(HeaderInertiaDevToolsID)
	if got := w.Header().Get(HeaderInertiaDevToolsParentOut); got != "root-entry" {
		t.Fatalf("unexpected parent-out %q", got)
	}
	entry := readDevToolsEntry(t, renderer, id)
	if entry.Meta.RequestType != "deferred" {
		t.Fatalf("unexpected request type %q", entry.Meta.RequestType)
	}
	if entry.Meta.BatchID == nil || *entry.Meta.BatchID != "root-entry" {
		t.Fatalf("unexpected batch id %#v", entry.Meta.BatchID)
	}
	if entry.Meta.TabUUID == nil || *entry.Meta.TabUUID != "tab-1" {
		t.Fatalf("unexpected tab id %#v", entry.Meta.TabUUID)
	}
	if entry.Meta.VisitID == nil || *entry.Meta.VisitID != "visit-1" {
		t.Fatalf("unexpected visit id %#v", entry.Meta.VisitID)
	}
	if got := entry.Props["stats"]; got.InertiaType != "defer" || got.DeferGroup != "analytics" {
		t.Fatalf("unexpected deferred metadata %#v", got)
	}
}

func TestDevToolsPrefetchStartsNewParent(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	req := newLocalDevToolsRequest(http.MethodGet, "/users", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderPurpose, "prefetch")
	req.Header.Set(HeaderInertiaDevToolsParent, "incoming-parent")
	w := httptest.NewRecorder()
	var renderErr error

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		renderErr = renderer.Render(w, req, "Users/Index", Props{})
	})).ServeHTTP(w, req)
	if renderErr != nil {
		t.Fatal(renderErr)
	}

	id := w.Header().Get(HeaderInertiaDevToolsID)
	if got := w.Header().Get(HeaderInertiaDevToolsParentOut); got != id {
		t.Fatalf("prefetch should return its own id as parent, got %q", got)
	}
	entry := readDevToolsEntry(t, renderer, id)
	if entry.Meta.RequestType != "prefetch" {
		t.Fatalf("unexpected request type %q", entry.Meta.RequestType)
	}
	if entry.Meta.BatchID == nil || *entry.Meta.BatchID != "incoming-parent" {
		t.Fatalf("unexpected batch id %#v", entry.Meta.BatchID)
	}
}

func TestDevToolsRedactsJSONRequestBody(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{
		Enabled:    true,
		RedactKeys: []string{"private_note"},
	}})
	body := `{"email":"ada@example.com","password":"secret","nested":{"token":"nested-secret","private_note":"hidden"}}`
	req := newLocalDevToolsRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	var readErr error

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, readErr = io.ReadAll(req.Body)
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(w, req)
	if readErr != nil {
		t.Fatal(readErr)
	}

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	requestBody := devToolsBodyMap(t, entry.HTTP.RequestBody)
	if requestBody["email"] != "ada@example.com" || requestBody["password"] != devToolsRedactedValue {
		t.Fatalf("unexpected request body %#v", requestBody)
	}
	nested, ok := requestBody["nested"].(map[string]any)
	if !ok || nested["token"] != devToolsRedactedValue || nested["private_note"] != devToolsRedactedValue {
		t.Fatalf("nested secrets were not redacted: %#v", requestBody["nested"])
	}
}

func TestDevToolsRedactsNestedFormRequestBody(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	body := "profile%5Bemail%5D=ada%40example.com&profile%5Bpassword%5D%5Bconfirmation%5D=form-secret"
	req := newLocalDevToolsRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = io.Copy(io.Discard, req.Body)
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(w, req)

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	requestBody := devToolsBodyMap(t, entry.HTTP.RequestBody)
	if requestBody["profile[email]"] != "ada@example.com" {
		t.Fatalf("non-sensitive nested form value changed: %#v", requestBody)
	}
	if requestBody["profile[password][confirmation]"] != devToolsRedactedValue {
		t.Fatalf("nested form secret was not redacted: %#v", requestBody)
	}
}

func TestDevToolsOmitsMalformedStructuredRequestBody(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	req := newLocalDevToolsRequest(http.MethodPost, "/users", strings.NewReader(`{"password":"secret"`))
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, _ = io.Copy(io.Discard, req.Body)
		w.WriteHeader(http.StatusBadRequest)
	})).ServeHTTP(w, req)

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	if entry.HTTP.RequestBody.Status != "omitted" || entry.HTTP.RequestBody.Reason != "unserializable" {
		t.Fatalf("unexpected malformed body capture %#v", entry.HTTP.RequestBody)
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "secret") {
		t.Fatalf("malformed structured body leaked into entry: %s", encoded)
	}
}

func TestDevToolsOmitsPartiallyReadRequestBody(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	req := newLocalDevToolsRequest(http.MethodPost, "/users", strings.NewReader(`{"password":"secret","name":"Ada"}`))
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		buffer := make([]byte, 8)
		_, _ = req.Body.Read(buffer)
		w.WriteHeader(http.StatusBadRequest)
	})).ServeHTTP(w, req)

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	if entry.HTTP.RequestBody.Status != "omitted" || entry.HTTP.RequestBody.Reason != "streamed" {
		t.Fatalf("unexpected partial body capture %#v", entry.HTTP.RequestBody)
	}
}

func TestDevToolsOmitsOverLimitRequestBody(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true, MaxBodyBytes: 8}})
	req := newLocalDevToolsRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"Ada"}`))
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
	})).ServeHTTP(w, req)

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	if entry.HTTP.RequestBody.Status != "omitted" || entry.HTTP.RequestBody.Reason != "too-large" {
		t.Fatalf("unexpected over-limit body capture %#v", entry.HTTP.RequestBody)
	}
}

func TestDevToolsRedactsSensitiveURLQueries(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	req := newLocalDevToolsRequest(http.MethodGet, "/dashboard?token=request-secret&filter%5Bapi_key%5D=nested-secret&profile%5Bpassword%5D%5Bconfirmation%5D=deep-secret&page=1", nil)
	req.Header.Set("Referer", "http://example.com/login?password=referer-secret&from=dashboard")
	w := httptest.NewRecorder()
	var renderErr error
	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		renderErr = renderer.Render(w, req, "Dashboard", Props{})
	})).ServeHTTP(w, req)
	if renderErr != nil {
		t.Fatal(renderErr)
	}

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	assertDevToolsRedactedQuery(t, entry.Meta.URL, "token")
	assertDevToolsRedactedQuery(t, entry.Meta.URL, "filter[api_key]")
	assertDevToolsRedactedQuery(t, entry.Meta.URL, "profile[password][confirmation]")
	parsedEntryURL, err := url.Parse(entry.Meta.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsedEntryURL.Query().Get("page"); got != "1" {
		t.Fatalf("non-sensitive query changed to %q", got)
	}
	page := devToolsBodyMap(t, entry.HTTP.ResponseBody)
	pageURL, ok := page["url"].(string)
	if !ok {
		t.Fatalf("unexpected page URL %#v", page["url"])
	}
	assertDevToolsRedactedQuery(t, pageURL, "token")
	assertDevToolsRedactedQuery(t, entry.HTTP.RequestHeaders["referer"], "password")
}

func TestDevToolsRedactsRedirectURLQueries(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	req := newLocalDevToolsRequest(http.MethodGet, "/redirect", nil)
	w := httptest.NewRecorder()
	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Location", "/next?token=redirect-secret&page=2")
		w.WriteHeader(http.StatusFound)
	})).ServeHTTP(w, req)

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	if entry.Meta.RedirectLocation == nil {
		t.Fatal("missing redirect location")
	}
	assertDevToolsRedactedQuery(t, *entry.Meta.RedirectLocation, "token")
	assertDevToolsRedactedQuery(t, entry.HTTP.ResponseHeaders["location"], "token")
}

func TestDevToolsSummarizesMultipartFiles(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("password", "form-secret"); err != nil {
		t.Fatal(err)
	}
	file, err := writer.CreateFormFile("avatar", "profile.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("private file contents")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := newLocalDevToolsRequest(http.MethodPost, "/profile", &body)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	var readErr error
	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, readErr = io.Copy(io.Discard, req.Body)
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(w, req)
	if readErr != nil {
		t.Fatal(readErr)
	}

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	requestBody := devToolsBodyMap(t, entry.HTTP.RequestBody)
	if requestBody["password"] != devToolsRedactedValue {
		t.Fatalf("multipart secret was not redacted: %#v", requestBody)
	}
	avatar, ok := requestBody["avatar"].(map[string]any)
	if !ok || avatar["name"] != "profile.txt" || avatar["size"] != float64(len("private file contents")) {
		t.Fatalf("unexpected file summary %#v", requestBody["avatar"])
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private file contents") || strings.Contains(string(encoded), "form-secret") {
		t.Fatalf("multipart contents leaked into entry: %s", encoded)
	}
}

func TestDevToolsOmitsNonTextualRequestBody(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	req := newLocalDevToolsRequest(http.MethodPost, "/upload", strings.NewReader("printable but binary content"))
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()
	var readErr error
	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_, readErr = io.Copy(io.Discard, req.Body)
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(w, req)
	if readErr != nil {
		t.Fatal(readErr)
	}

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	if entry.HTTP.RequestBody.Status != "omitted" || entry.HTTP.RequestBody.Reason != "non-textual" {
		t.Fatalf("unexpected request body capture %#v", entry.HTTP.RequestBody)
	}
}

func TestDevToolsClassifiesNestedComposableProps(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{
		SharedProps: StaticSharedProps(Props{
			"auth": Props{
				"permissions": Optional([]string{"users.read"}),
			},
		}),
		DevTools: DevToolsConfig{Enabled: true},
	})
	req := newLocalDevToolsRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	var renderErr error
	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		renderErr = renderer.Render(w, req, "Dashboard", Props{
			"activity": Computed(func(*http.Request) (any, error) {
				return []string{"signed-in"}, nil
			}).Defer("activity").DeepMerge().Once(),
		})
	})).ServeHTTP(w, req)
	if renderErr != nil {
		t.Fatal(renderErr)
	}

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	permissions := entry.Props["auth.permissions"]
	if permissions.InertiaType != "optional" || !permissions.Shared {
		t.Fatalf("unexpected nested optional metadata %#v", permissions)
	}
	activity := entry.Props["activity"]
	if activity.InertiaType != "defer" || activity.DeferGroup != "activity" || !activity.Once || !activity.DeepMerge || activity.MergeDirection != "append" {
		t.Fatalf("unexpected composable metadata %#v", activity)
	}
}

func TestDevToolsRecordsNestedPropValues(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	req := newLocalDevToolsRequest(http.MethodGet, "/dashboard", nil)
	req.Header.Set(HeaderInertia, "true")
	req.Header.Set(HeaderInertiaPartialComponent, "Dashboard")
	req.Header.Set(HeaderInertiaPartialData, "auth.permissions")
	w := httptest.NewRecorder()
	var renderErr error
	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		renderErr = renderer.Render(w, req, "Dashboard", Props{
			"auth": Props{
				"permissions": Optional([]string{"users.read"}),
			},
		})
	})).ServeHTTP(w, req)
	if renderErr != nil {
		t.Fatal(renderErr)
	}

	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	permissions, ok := entry.PropValues["auth.permissions"].([]any)
	if !ok || len(permissions) != 1 || permissions[0] != "users.read" {
		t.Fatalf("unexpected nested prop value %#v", entry.PropValues["auth.permissions"])
	}
}

func TestDevToolsAuthorizationDefaultsToLoopback(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/dashboard", nil)
	req.RemoteAddr = "198.51.100.10:12345"
	w := httptest.NewRecorder()

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(w, req)
	if got := w.Code; got != http.StatusNoContent {
		t.Fatalf("unexpected application response status %d", got)
	}
	if got := w.Header().Get(HeaderInertiaDevToolsID); got != "" {
		t.Fatalf("unauthorized request was recorded as %q", got)
	}

	endpointReq := httptest.NewRequest(http.MethodGet, "http://example.com"+devToolsEntriesPath, nil)
	endpointReq.RemoteAddr = "198.51.100.10:12345"
	endpointResponse := httptest.NewRecorder()
	renderer.Middleware(http.NotFoundHandler()).ServeHTTP(endpointResponse, endpointReq)
	if got := endpointResponse.Code; got != http.StatusForbidden {
		t.Fatalf("unexpected endpoint status %d", got)
	}
}

func TestDevToolsAuthorizationPanicDoesNotBreakApplication(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{
		Enabled: true,
		Authorize: func(*http.Request) bool {
			panic("authorization failed")
		},
	}})
	req := newLocalDevToolsRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(w, req)

	if got := w.Code; got != http.StatusNoContent {
		t.Fatalf("unexpected application response status %d", got)
	}
	if got := w.Header().Get(HeaderInertiaDevToolsID); got != "" {
		t.Fatalf("request with panicking authorization was recorded as %q", got)
	}
}

func TestDevToolsDisabledLeavesEndpointToApplication(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{})
	req := newLocalDevToolsRequest(http.MethodGet, devToolsEntriesPath, nil)
	w := httptest.NewRecorder()

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})).ServeHTTP(w, req)
	if got := w.Code; got != http.StatusTeapot {
		t.Fatalf("disabled endpoint did not reach application: %d", got)
	}
	if got := w.Header().Get(HeaderInertiaDevToolsID); got != "" {
		t.Fatalf("disabled recorder emitted id %q", got)
	}
}

func TestDevToolsStoreEnforcesPerTabLimit(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true, Limit: 1}})
	handler := renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	firstReq := newLocalDevToolsRequest(http.MethodGet, "/first", nil)
	firstReq.Header.Set(HeaderInertiaDevToolsTab, "tab-1")
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, firstReq)
	firstID := firstResponse.Header().Get(HeaderInertiaDevToolsID)

	secondReq := newLocalDevToolsRequest(http.MethodGet, "/second", nil)
	secondReq.Header.Set(HeaderInertiaDevToolsTab, "tab-1")
	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, secondReq)
	secondID := secondResponse.Header().Get(HeaderInertiaDevToolsID)

	if status := devToolsEntryStatus(t, renderer, firstID); status != http.StatusNotFound {
		t.Fatalf("oldest same-tab entry should be evicted, got %d", status)
	}
	if status := devToolsEntryStatus(t, renderer, secondID); status != http.StatusOK {
		t.Fatalf("newest same-tab entry should remain, got %d", status)
	}
}

func TestDevToolsStorePrunesExpiredEntries(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newDevToolsStore(10, time.Minute)
	state := &devToolsRequestState{id: "expired"}
	store.add(state, now)

	if _, ok := store.get(state.id, now.Add(time.Minute+time.Nanosecond)); ok {
		t.Fatal("expired entry remained in the store")
	}
}

func TestDevToolsReadAPIListsNewestAndFiltersEntries(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	record := func(path string, component string, headers map[string]string) string {
		t.Helper()
		req := newLocalDevToolsRequest(http.MethodGet, path, nil)
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		w := httptest.NewRecorder()
		var renderErr error
		renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			renderErr = renderer.Render(w, req, component, Props{})
		})).ServeHTTP(w, req)
		if renderErr != nil {
			t.Fatal(renderErr)
		}
		return w.Header().Get(HeaderInertiaDevToolsID)
	}

	record("/dashboard", "Dashboard", nil)
	record("/users", "Users/Index", map[string]string{HeaderInertia: "true"})
	pollID := record("/dashboard", "Dashboard", map[string]string{
		HeaderInertia:             "true",
		HeaderInertiaDevToolsPoll: "1",
	})

	entries, response := readDevToolsEntries(t, renderer, "?component=Dashboard&type=poll,initial&exclude=initial&limit=1")
	if len(entries) != 1 || entries[0].Meta.ID != pollID || entries[0].Meta.RequestType != "poll" {
		t.Fatalf("unexpected filtered entries %#v", entries)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("unexpected cache control %q", got)
	}
	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("unexpected content type protection %q", got)
	}

	methodRequest := newLocalDevToolsRequest(http.MethodPost, devToolsEntriesPath, strings.NewReader("ignored"))
	methodResponse := httptest.NewRecorder()
	renderer.Middleware(http.NotFoundHandler()).ServeHTTP(methodResponse, methodRequest)
	if got := methodResponse.Code; got != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected method status %d", got)
	}
	if got := methodResponse.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("unexpected allow header %q", got)
	}
}

func TestDevToolsRequestTypePrecedence(t *testing.T) {
	tests := []struct {
		name  string
		state *devToolsRequestState
		want  string
	}{
		{name: "precognition", state: &devToolsRequestState{isPrecognition: true, isInertia: true, isDeferred: true}, want: "precognition"},
		{name: "initial", state: &devToolsRequestState{component: nonEmptyString("Dashboard")}, want: "initial"},
		{name: "http", state: &devToolsRequestState{}, want: "http"},
		{name: "deferred", state: &devToolsRequestState{isInertia: true, isDeferred: true, isPoll: true}, want: "deferred"},
		{name: "poll", state: &devToolsRequestState{isInertia: true, isPoll: true, isPartial: true}, want: "poll"},
		{name: "partial", state: &devToolsRequestState{isInertia: true, isPartial: true, isPrefetch: true}, want: "partial"},
		{name: "prefetch", state: &devToolsRequestState{isInertia: true, isPrefetch: true}, want: "prefetch"},
		{name: "navigate", state: &devToolsRequestState{isInertia: true}, want: "navigate"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.state.requestType(); got != test.want {
				t.Fatalf("requestType() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDevToolsStoredEntryTracksLateErrorResponse(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{Enabled: true}})
	req := newLocalDevToolsRequest(http.MethodGet, "/failure", nil)
	w := httptest.NewRecorder()
	var capturedRequest *http.Request
	var capturedWriter http.ResponseWriter

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		capturedRequest = req
		capturedWriter = w
	})).ServeHTTP(w, req)
	id := w.Header().Get(HeaderInertiaDevToolsID)

	if err := renderer.RenderError(capturedWriter, capturedRequest, "Error", Props{"message": "failed"}, http.StatusInternalServerError); err != nil {
		t.Fatal(err)
	}
	entry := readDevToolsEntry(t, renderer, id)
	if entry.Meta.Status != http.StatusInternalServerError {
		t.Fatalf("late status was not recorded: %#v", entry.Meta)
	}
	if entry.Meta.Component == nil || *entry.Meta.Component != "Error" {
		t.Fatalf("late component was not recorded: %#v", entry.Meta.Component)
	}
	if entry.RenderSource == nil || !strings.HasSuffix(entry.RenderSource.File, "devtools_test.go") || entry.RenderSource.Line < 1 {
		t.Fatalf("unexpected error render source %#v", entry.RenderSource)
	}
}

func TestDevToolsComponentResolverPanicDoesNotBreakResponse(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{DevTools: DevToolsConfig{
		Enabled: true,
		ComponentPathResolver: func(string) string {
			panic("resolver failed")
		},
	}})
	req := newLocalDevToolsRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	var renderErr error

	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		renderErr = renderer.Render(w, req, "Dashboard", Props{})
	})).ServeHTTP(w, req)
	if renderErr != nil {
		t.Fatal(renderErr)
	}
	if got := w.Code; got != http.StatusOK {
		t.Fatalf("unexpected response status %d", got)
	}
	entry := readDevToolsEntry(t, renderer, w.Header().Get(HeaderInertiaDevToolsID))
	if entry.ComponentPath != nil {
		t.Fatalf("panicking resolver should leave component path nil: %#v", entry.ComponentPath)
	}
}

func TestDevToolsSerializationPanicDoesNotBreakResponse(t *testing.T) {
	renderer := newDevToolsTestRenderer(t, Config{
		JSONEncoder: devToolsStaticJSONEncoder{},
		DevTools:    DevToolsConfig{Enabled: true},
	})
	req := newLocalDevToolsRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()
	var renderErr error
	renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		renderErr = renderer.Render(w, req, "Dashboard", Props{"unsafe": devToolsPanickingJSONValue{}})
	})).ServeHTTP(w, req)
	if renderErr != nil {
		t.Fatal(renderErr)
	}
	if got := w.Code; got != http.StatusOK {
		t.Fatalf("unexpected response status %d", got)
	}
	if got := w.Header().Get(HeaderInertiaDevToolsID); got == "" {
		t.Fatal("missing entry id after recorder failure")
	}
}

func TestDevToolsNestedPropClassificationStopsAtCycles(t *testing.T) {
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	req := newLocalDevToolsRequest(http.MethodGet, "/dashboard", nil)
	props := map[string]devToolsPropMeta{}

	classifyNestedDevToolsProps(req, "cyclic", cyclic, false, props)

	if len(props) != 0 {
		t.Fatalf("unexpected metadata from cyclic value %#v", props)
	}
}

func TestInjectDevToolsIDScript(t *testing.T) {
	script := []byte(`<script data-inertia-devtools-id type="application/json">"entry"</script>`)
	tests := []struct {
		name string
		body string
	}{
		{name: "lowercase body", body: `<html><body>app</body></html>`},
		{name: "uppercase body", body: `<HTML><BODY>app</BODY></HTML>`},
		{name: "fragment", body: `<div>app</div>`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := injectDevToolsIDScript([]byte(test.body), script)
			if count := bytes.Count(got, script); count != 1 {
				t.Fatalf("unexpected discovery script count %d in %s", count, got)
			}
		})
	}
}

func newDevToolsTestRenderer(t *testing.T, config Config) *Renderer {
	t.Helper()
	if config.RootView == nil {
		view := template.Must(template.New("app").Parse(`<!doctype html><html><body>{{ .InertiaApp }}</body></html>`))
		config.RootView = NewTemplateRootView(view, "app")
	}
	renderer, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	return renderer
}

func newLocalDevToolsRequest(method string, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, "http://example.com"+target, body)
	req.RemoteAddr = "127.0.0.1:12345"
	return req
}

func readDevToolsEntry(t *testing.T, renderer *Renderer, id string) devToolsEntry {
	t.Helper()
	path := devToolsEntriesPath + "/" + id
	req := newLocalDevToolsRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	renderer.Middleware(http.NotFoundHandler()).ServeHTTP(w, req)
	if got := w.Code; got != http.StatusOK {
		t.Fatalf("unexpected entry endpoint status %d: %s", got, w.Body.String())
	}
	var entry devToolsEntry
	if err := json.Unmarshal(w.Body.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	return entry
}

func devToolsEntryStatus(t *testing.T, renderer *Renderer, id string) int {
	t.Helper()
	path := devToolsEntriesPath + "/" + id
	req := newLocalDevToolsRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	renderer.Middleware(http.NotFoundHandler()).ServeHTTP(w, req)
	return w.Code
}

func readDevToolsEntries(t *testing.T, renderer *Renderer, query string) ([]devToolsEntry, *httptest.ResponseRecorder) {
	t.Helper()
	req := newLocalDevToolsRequest(http.MethodGet, devToolsEntriesPath+query, nil)
	w := httptest.NewRecorder()
	renderer.Middleware(http.NotFoundHandler()).ServeHTTP(w, req)
	if got := w.Code; got != http.StatusOK {
		t.Fatalf("unexpected entries endpoint status %d: %s", got, w.Body.String())
	}
	var entries []devToolsEntry
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	return entries, w
}

func devToolsBodyMap(t *testing.T, capture devToolsBodyCapture) map[string]any {
	t.Helper()
	value, ok := capture.Value.(map[string]any)
	if !ok {
		t.Fatalf("unexpected body value %#v", capture.Value)
	}
	return value
}

func assertDevToolsRedactedQuery(t *testing.T, value string, key string) {
	t.Helper()
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Query().Get(key); got != devToolsRedactedValue {
		t.Fatalf("query %q was not redacted in %q: %q", key, value, got)
	}
}

type devToolsStaticJSONEncoder struct{}

func (devToolsStaticJSONEncoder) Encode(any) ([]byte, error) {
	return []byte(`{"component":"Dashboard","props":{},"url":"/dashboard","version":""}`), nil
}

type devToolsPanickingJSONValue struct{}

func (devToolsPanickingJSONValue) MarshalJSON() ([]byte, error) {
	panic("serialization failed")
}

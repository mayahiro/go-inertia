package inertia

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// HeaderInertiaDevToolsID identifies the recorded entry for a response.
	HeaderInertiaDevToolsID = "X-Inertia-Devtools-Id"
	// HeaderInertiaDevToolsTab identifies the browser tab that issued a request.
	HeaderInertiaDevToolsTab = "X-Inertia-Devtools-Tab"
	// HeaderInertiaDevToolsVisit correlates a request with a client-side visit.
	HeaderInertiaDevToolsVisit = "X-Inertia-Devtools-Visit"
	// HeaderInertiaDevToolsParent identifies the root entry of a request batch.
	HeaderInertiaDevToolsParent = "X-Inertia-Devtools-Parent"
	// HeaderInertiaDevToolsDeferred marks a deferred-prop follow-up request.
	HeaderInertiaDevToolsDeferred = "X-Inertia-Devtools-Deferred"
	// HeaderInertiaDevToolsPoll marks a polling request.
	HeaderInertiaDevToolsPoll = "X-Inertia-Devtools-Poll"
	// HeaderInertiaDevToolsParentOut carries the batch root to the client.
	HeaderInertiaDevToolsParentOut = "X-Inertia-Devtools-Parent-Out"

	devToolsEntriesPath   = "/_inertia/devtools/entries"
	devToolsDefaultLimit  = 100
	devToolsDefaultBody   = 256_000
	devToolsMaximumIDSize = 256
)

var errDevToolsID = errors.New("inertia: could not generate devtools entry id")

// DevToolsAuthorizeFunc reports whether a request may be recorded and may read
// recorded entries.
type DevToolsAuthorizeFunc func(req *http.Request) bool

// DevToolsComponentPathResolver returns the frontend source path for component.
// An empty result leaves the optional componentPath entry field null.
type DevToolsComponentPathResolver func(component string) string

// DevToolsConfig configures the backend-independent Inertia DevTools recorder.
// DevTools is disabled unless Enabled is true.
type DevToolsConfig struct {
	// Enabled enables response discovery headers, recording, and the read API.
	Enabled bool
	// Authorize controls both recording and read API access. When nil, only
	// requests whose direct remote address is loopback are authorized.
	Authorize DevToolsAuthorizeFunc
	// Limit is the maximum number of entries kept per browser tab. Values below
	// one use the default of 100.
	Limit int
	// TTL controls how long entries remain readable. Values below one use 24 hours.
	TTL time.Duration
	// MaxBodyBytes limits captured request bodies. Values below one use 256000 bytes.
	MaxBodyBytes int
	// RedactKeys adds case-insensitive JSON and form keys to the default secret list.
	RedactKeys []string
	// RedactHeaders adds case-insensitive HTTP headers to the default secret list.
	RedactHeaders []string
	// ComponentPathResolver optionally maps an Inertia component to its frontend file.
	ComponentPathResolver DevToolsComponentPathResolver
}

type devToolsSource struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

type devToolsRecorder struct {
	authorize             DevToolsAuthorizeFunc
	componentPathResolver DevToolsComponentPathResolver
	maxBodyBytes          int
	redactor              devToolsRedactor
	store                 *devToolsStore
}

type devToolsContextKey struct{}

func newDevToolsRecorder(config DevToolsConfig) *devToolsRecorder {
	if !config.Enabled {
		return nil
	}

	authorize := config.Authorize
	if authorize == nil {
		authorize = authorizeLoopbackDevTools
	}
	limit := config.Limit
	if limit < 1 {
		limit = devToolsDefaultLimit
	}
	ttl := config.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	maxBodyBytes := config.MaxBodyBytes
	if maxBodyBytes < 1 {
		maxBodyBytes = devToolsDefaultBody
	}

	return &devToolsRecorder{
		authorize:             authorize,
		componentPathResolver: config.ComponentPathResolver,
		maxBodyBytes:          maxBodyBytes,
		redactor:              newDevToolsRedactor(config.RedactKeys, config.RedactHeaders),
		store:                 newDevToolsStore(limit, ttl),
	}
}

func authorizeLoopbackDevTools(req *http.Request) bool {
	if req == nil {
		return false
	}
	host := req.RemoteAddr
	if parsedHost, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (d *devToolsRecorder) authorized(req *http.Request) (allowed bool) {
	defer func() {
		if recover() != nil {
			allowed = false
		}
	}()
	return d.authorize(req)
}

func (d *devToolsRecorder) isEndpoint(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	return req.URL.Path == devToolsEntriesPath || strings.HasPrefix(req.URL.Path, devToolsEntriesPath+"/")
}

func (d *devToolsRecorder) serveEndpoint(w http.ResponseWriter, req *http.Request) {
	if !d.authorized(req) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if req.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if req.URL.Path == devToolsEntriesPath {
		entries := filterDevToolsEntries(d.store.all(time.Now()), req)
		if err := json.NewEncoder(w).Encode(entries); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	id := strings.TrimPrefix(req.URL.Path, devToolsEntriesPath+"/")
	if id == "" || len(id) > devToolsMaximumIDSize || strings.Contains(id, "/") {
		http.NotFound(w, req)
		return
	}
	entry, ok := d.store.get(id, time.Now())
	if !ok {
		http.NotFound(w, req)
		return
	}
	if err := json.NewEncoder(w).Encode(entry); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func filterDevToolsEntries(entries []devToolsEntry, req *http.Request) []devToolsEntry {
	component := req.URL.Query().Get("component")
	types := splitHeaderList(req.URL.Query().Get("type"))
	excluded := splitHeaderList(req.URL.Query().Get("exclude"))
	filtered := make([]devToolsEntry, 0, len(entries))
	for _, entry := range entries {
		if component != "" && (entry.Meta.Component == nil || *entry.Meta.Component != component) {
			continue
		}
		if len(types) > 0 && !containsString(types, entry.Meta.RequestType) {
			continue
		}
		if containsString(excluded, entry.Meta.RequestType) {
			continue
		}
		filtered = append(filtered, entry)
	}

	offset, _ := strconv.Atoi(req.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	if offset >= len(filtered) {
		return []devToolsEntry{}
	}
	filtered = filtered[offset:]
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}
	return filtered
}

func (d *devToolsRecorder) begin(w http.ResponseWriter, req *http.Request) (http.ResponseWriter, *http.Request, *devToolsRequestState) {
	id, err := newDevToolsID()
	if err != nil {
		return w, req, nil
	}

	var batchID *string
	if IsInertiaRequest(req) {
		batchID = nonEmptyString(req.Header.Get(HeaderInertiaDevToolsParent))
	}
	parentOut := id
	if !IsPrefetch(req) && batchID != nil {
		parentOut = *batchID
	}

	w.Header().Set(HeaderInertiaDevToolsID, id)
	w.Header().Set(HeaderInertiaDevToolsParentOut, parentOut)

	state := &devToolsRequestState{
		id:               id,
		tabUUID:          nonEmptyString(req.Header.Get(HeaderInertiaDevToolsTab)),
		batchID:          batchID,
		visitID:          nonEmptyString(req.Header.Get(HeaderInertiaDevToolsVisit)),
		startedAt:        time.Now(),
		method:           req.Method,
		url:              d.redactor.url(absoluteRequestURL(req)),
		isInertia:        IsInertiaRequest(req),
		isPrecognition:   hasDevToolsHeader(req, HeaderPrecognition),
		isDeferred:       hasDevToolsHeader(req, HeaderInertiaDevToolsDeferred),
		isPoll:           hasDevToolsHeader(req, HeaderInertiaDevToolsPoll),
		isPartial:        hasDevToolsHeader(req, HeaderInertiaPartialComponent),
		isPrefetch:       IsPrefetch(req),
		requestHeaders:   d.redactor.headers(req.Header),
		requestBody:      devToolsEmptyBody(),
		responseBody:     devToolsEmptyBody(),
		props:            map[string]devToolsPropMeta{},
		propValues:       map[string]any{},
		responseHeaders:  map[string]string{},
		componentPath:    nil,
		responseStatus:   0,
		redirectLocation: nil,
	}

	requestWithState := req.WithContext(context.WithValue(req.Context(), devToolsContextKey{}, state))
	if req.Body != nil {
		capture := newDevToolsBodyReader(req.Body, d.maxBodyBytes)
		requestWithState.Body = capture
		state.bodyReader = capture
	}

	writer := &devToolsResponseWriter{
		ResponseWriter: w,
		state:          state,
		redactor:       &d.redactor,
	}
	return writer, requestWithState, state
}

func (d *devToolsRecorder) finish(state *devToolsRequestState, req *http.Request, writer *devToolsResponseWriter) {
	defer func() {
		_ = recover()
	}()
	state.finishRequest(req, d.maxBodyBytes, d.redactor)
	state.finishResponse(writer.Header(), d.redactor)
	d.store.add(state, time.Now())
}

func newDevToolsID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", errDevToolsID
	}
	return hex.EncodeToString(value[:]), nil
}

func nonEmptyString(value string) *string {
	if value == "" {
		return nil
	}
	cloned := value
	return &cloned
}

func absoluteRequestURL(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}
	if req.URL.IsAbs() {
		return req.URL.String()
	}
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + req.Host + req.URL.RequestURI()
}

func devToolsState(req *http.Request) *devToolsRequestState {
	if req == nil {
		return nil
	}
	state, _ := req.Context().Value(devToolsContextKey{}).(*devToolsRequestState)
	return state
}

func hasDevToolsHeader(req *http.Request, name string) bool {
	if req == nil {
		return false
	}
	for key := range req.Header {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}

// RecordDevToolsRoute attaches framework route metadata to the current recorded
// request. Empty name and action values are encoded as null. The method is a
// no-op when DevTools is disabled or req did not pass through Renderer.Middleware.
func (r *Renderer) RecordDevToolsRoute(req *http.Request, name string, uri string, action string) {
	if r == nil || r.devTools == nil {
		return
	}
	state := devToolsState(req)
	if state == nil {
		return
	}
	state.setRoute(devToolsRoute{Name: name, URI: uri, Action: action})
}

// RecordDevToolsRenderSource attaches the user-space render call site to the
// current recorded request. It is intended for framework adapters.
func (r *Renderer) RecordDevToolsRenderSource(req *http.Request, file string, line int) {
	if r == nil || r.devTools == nil {
		return
	}
	state := devToolsState(req)
	if state == nil || file == "" || line < 1 {
		return
	}
	state.setRenderSource(devToolsSource{File: file, Line: line}, true)
}

type devToolsResponseWriter struct {
	http.ResponseWriter
	state       *devToolsRequestState
	redactor    *devToolsRedactor
	wroteHeader bool
}

func (w *devToolsResponseWriter) WriteHeader(status int) {
	final := status >= 200 || status == http.StatusSwitchingProtocols
	if final && !w.wroteHeader {
		w.wroteHeader = true
		w.state.recordResponse(status, w.redactor.headers(w.Header()), time.Now(), *w.redactor)
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *devToolsResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *devToolsResponseWriter) ReadFrom(reader io.Reader) (int64, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return io.Copy(w.ResponseWriter, reader)
}

func (w *devToolsResponseWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *devToolsResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

func (w *devToolsResponseWriter) Push(target string, options *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, options)
}

func (w *devToolsResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type devToolsStore struct {
	mu      sync.Mutex
	entries map[string]*devToolsRequestState
	order   []string
	limit   int
	ttl     time.Duration
}

func newDevToolsStore(limit int, ttl time.Duration) *devToolsStore {
	return &devToolsStore{
		entries: map[string]*devToolsRequestState{},
		limit:   limit,
		ttl:     ttl,
	}
}

func (s *devToolsStore) add(state *devToolsRequestState, now time.Time) {
	state.markRecorded(now)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	if _, exists := s.entries[state.id]; exists {
		return
	}
	s.entries[state.id] = state
	s.order = append(s.order, state.id)
	s.enforceTabLimitLocked(state.tabKey())
}

func (s *devToolsStore) get(id string, now time.Time) (devToolsEntry, bool) {
	s.mu.Lock()
	s.pruneLocked(now)
	state, ok := s.entries[id]
	s.mu.Unlock()
	if !ok {
		return devToolsEntry{}, false
	}
	return state.entry(), true
}

func (s *devToolsStore) all(now time.Time) []devToolsEntry {
	s.mu.Lock()
	s.pruneLocked(now)
	states := make([]*devToolsRequestState, 0, len(s.order))
	for index := len(s.order) - 1; index >= 0; index-- {
		states = append(states, s.entries[s.order[index]])
	}
	s.mu.Unlock()

	entries := make([]devToolsEntry, 0, len(states))
	for _, state := range states {
		entries = append(entries, state.entry())
	}
	return entries
}

func (s *devToolsStore) pruneLocked(now time.Time) {
	kept := s.order[:0]
	for _, id := range s.order {
		state, ok := s.entries[id]
		if !ok {
			continue
		}
		if now.Sub(state.recordedTime()) > s.ttl {
			delete(s.entries, id)
			continue
		}
		kept = append(kept, id)
	}
	s.order = kept
}

func (s *devToolsStore) enforceTabLimitLocked(tab string) {
	count := 0
	remove := map[string]bool{}
	for index := len(s.order) - 1; index >= 0; index-- {
		id := s.order[index]
		state := s.entries[id]
		if state == nil || state.tabKey() != tab {
			continue
		}
		count++
		if count > s.limit {
			remove[id] = true
			delete(s.entries, id)
		}
	}
	if len(remove) == 0 {
		return
	}
	kept := s.order[:0]
	for _, id := range s.order {
		if !remove[id] {
			kept = append(kept, id)
		}
	}
	s.order = kept
}

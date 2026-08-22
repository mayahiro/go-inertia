package inertia

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const devToolsRedactedValue = "[REDACTED]"

const devToolsMaximumPropDepth = 64

var defaultDevToolsRedactKeys = []string{
	"password",
	"password_confirmation",
	"current_password",
	"token",
	"_token",
	"access_token",
	"refresh_token",
	"secret",
	"client_secret",
	"api_key",
}

var defaultDevToolsRedactHeaders = []string{
	"cookie",
	"set-cookie",
	"authorization",
	"proxy-authorization",
	"x-xsrf-token",
	"x-csrf-token",
}

type devToolsEntry struct {
	Meta          devToolsEntryMeta           `json:"__meta"`
	HTTP          devToolsHTTP                `json:"http"`
	Props         map[string]devToolsPropMeta `json:"props"`
	PropValues    map[string]any              `json:"propValues"`
	Route         devToolsEntryRoute          `json:"route"`
	RenderSource  *devToolsSource             `json:"renderSource"`
	ComponentPath *string                     `json:"componentPath"`
}

type devToolsEntryMeta struct {
	ID               string  `json:"id"`
	TabUUID          *string `json:"tabUuid"`
	BatchID          *string `json:"batchId"`
	Timestamp        string  `json:"timestamp"`
	Utime            float64 `json:"utime"`
	Method           string  `json:"method"`
	URL              string  `json:"url"`
	Component        *string `json:"component"`
	RequestType      string  `json:"requestType"`
	Status           int     `json:"status"`
	RedirectLocation *string `json:"redirectLocation"`
	ServerTimingMS   float64 `json:"serverTimingMs"`
	VisitID          *string `json:"visitId"`
}

type devToolsHTTP struct {
	RequestHeaders  map[string]string   `json:"requestHeaders"`
	ResponseHeaders map[string]string   `json:"responseHeaders"`
	RequestBody     devToolsBodyCapture `json:"requestBody"`
	ResponseBody    devToolsBodyCapture `json:"responseBody"`
}

type devToolsBodyCapture struct {
	Status string `json:"status"`
	Value  any    `json:"value,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type devToolsPropMeta struct {
	InertiaType    string `json:"inertiaType,omitempty"`
	Shared         bool   `json:"shared"`
	DeferGroup     string `json:"deferGroup,omitempty"`
	Reset          bool   `json:"reset,omitempty"`
	Once           bool   `json:"once,omitempty"`
	MergeDirection string `json:"mergeDirection,omitempty"`
	DeepMerge      bool   `json:"deepMerge,omitempty"`
	Rescued        bool   `json:"rescued,omitempty"`
}

type devToolsEntryRoute struct {
	Name         *string         `json:"name"`
	URI          string          `json:"uri"`
	Action       *string         `json:"action"`
	ActionSource *devToolsSource `json:"actionSource,omitempty"`
}

type devToolsRoute struct {
	Name         string
	URI          string
	Action       string
	ActionSource *devToolsSource
}

type devToolsRequestState struct {
	mu sync.RWMutex

	id      string
	tabUUID *string
	batchID *string
	visitID *string

	startedAt   time.Time
	recordedAt  time.Time
	respondedAt time.Time

	method string
	url    string

	isInertia      bool
	isPrecognition bool
	isDeferred     bool
	isPoll         bool
	isPartial      bool
	isPrefetch     bool

	requestHeaders  map[string]string
	responseHeaders map[string]string
	requestBody     devToolsBodyCapture
	responseBody    devToolsBodyCapture
	bodyReader      *devToolsBodyReader

	component        *string
	responseStatus   int
	redirectLocation *string
	props            map[string]devToolsPropMeta
	propValues       map[string]any
	route            devToolsRoute
	renderSource     *devToolsSource
	componentPath    *string
}

func devToolsEmptyBody() devToolsBodyCapture {
	return devToolsBodyCapture{Status: "empty"}
}

func devToolsOmittedBody(reason string) devToolsBodyCapture {
	return devToolsBodyCapture{Status: "omitted", Reason: reason}
}

func devToolsPresentBody(value any) devToolsBodyCapture {
	return devToolsBodyCapture{Status: "present", Value: value}
}

func (s *devToolsRequestState) recordResponse(status int, headers map[string]string, respondedAt time.Time, redactor devToolsRedactor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseStatus = status
	s.responseHeaders = headers
	s.respondedAt = respondedAt
	s.redirectLocation = redactor.urlPointer(redirectLocation(headers, status))
}

func (s *devToolsRequestState) finishRequest(req *http.Request, maxBodyBytes int, redactor devToolsRedactor) {
	body := captureDevToolsRequestBody(req, s.bodyReader, maxBodyBytes, redactor)
	s.mu.Lock()
	s.requestBody = body
	s.bodyReader = nil
	s.mu.Unlock()
}

func (s *devToolsRequestState) finishResponse(headers http.Header, redactor devToolsRedactor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.responseHeaders) == 0 {
		s.responseHeaders = redactor.headers(headers)
	}
	if s.responseStatus == 0 {
		s.redirectLocation = redactor.urlPointer(redirectLocation(s.responseHeaders, http.StatusOK))
	}
}

func (s *devToolsRequestState) markRecorded(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recordedAt = now
	if s.respondedAt.IsZero() {
		s.respondedAt = now
	}
}

func (s *devToolsRequestState) setRoute(route devToolsRoute) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.route = route
}

func (s *devToolsRequestState) setRenderSource(source devToolsSource, overwrite bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.renderSource != nil && !overwrite {
		return
	}
	cloned := source
	s.renderSource = &cloned
}

func (s *devToolsRequestState) setPage(
	component string,
	responseBody devToolsBodyCapture,
	props map[string]devToolsPropMeta,
	propValues map[string]any,
	componentPath *string,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.component = nonEmptyString(component)
	s.responseBody = responseBody
	s.props = props
	s.propValues = propValues
	s.componentPath = componentPath
}

func (s *devToolsRequestState) entry() devToolsEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := s.responseStatus
	if status == 0 {
		status = http.StatusOK
	}
	responseBody := s.responseBody
	if s.component == nil && !s.isInertia && responseBody.Status == "empty" {
		responseBody = devToolsOmittedBody("non-inertia-response")
	}
	timestamp := s.recordedAt.UTC()
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	respondedAt := s.respondedAt
	if respondedAt.IsZero() {
		respondedAt = timestamp
	}

	return devToolsEntry{
		Meta: devToolsEntryMeta{
			ID:               s.id,
			TabUUID:          cloneStringPointer(s.tabUUID),
			BatchID:          cloneStringPointer(s.batchID),
			Timestamp:        timestamp.Format("2006-01-02T15:04:05.000Z"),
			Utime:            float64(timestamp.UnixNano()) / float64(time.Second),
			Method:           s.method,
			URL:              s.url,
			Component:        cloneStringPointer(s.component),
			RequestType:      s.requestType(),
			Status:           status,
			RedirectLocation: cloneStringPointer(s.redirectLocation),
			ServerTimingMS:   float64(respondedAt.Sub(s.startedAt).Nanoseconds()) / float64(time.Millisecond),
			VisitID:          cloneStringPointer(s.visitID),
		},
		HTTP: devToolsHTTP{
			RequestHeaders:  cloneStringMap(s.requestHeaders),
			ResponseHeaders: cloneStringMap(s.responseHeaders),
			RequestBody:     s.requestBody,
			ResponseBody:    responseBody,
		},
		Props:         cloneDevToolsProps(s.props),
		PropValues:    cloneAnyMap(s.propValues),
		Route:         devToolsRouteEntry(s.route),
		RenderSource:  cloneDevToolsSource(s.renderSource),
		ComponentPath: cloneStringPointer(s.componentPath),
	}
}

func (s *devToolsRequestState) requestType() string {
	if s.isPrecognition {
		return "precognition"
	}
	if !s.isInertia {
		if s.component != nil {
			return "initial"
		}
		return "http"
	}
	if s.isDeferred {
		return "deferred"
	}
	if s.isPoll {
		return "poll"
	}
	if s.isPartial {
		return "partial"
	}
	if s.isPrefetch {
		return "prefetch"
	}
	return "navigate"
}

func (s *devToolsRequestState) tabKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.tabUUID == nil {
		return ""
	}
	return *s.tabUUID
}

func (s *devToolsRequestState) recordedTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recordedAt
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneDevToolsSource(source *devToolsSource) *devToolsSource {
	if source == nil {
		return nil
	}
	cloned := *source
	return &cloned
}

func cloneStringMap(source map[string]string) map[string]string {
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneAnyMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneDevToolsProps(source map[string]devToolsPropMeta) map[string]devToolsPropMeta {
	cloned := make(map[string]devToolsPropMeta, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func devToolsRouteEntry(route devToolsRoute) devToolsEntryRoute {
	return devToolsEntryRoute{
		Name:         nonEmptyString(route.Name),
		URI:          route.URI,
		Action:       nonEmptyString(route.Action),
		ActionSource: cloneDevToolsSource(route.ActionSource),
	}
}

func redirectLocation(headers map[string]string, status int) *string {
	if value := headers[strings.ToLower(HeaderInertiaLocation)]; value != "" {
		return nonEmptyString(value)
	}
	if status < 300 || status >= 400 {
		return nil
	}
	return nonEmptyString(headers["location"])
}

type devToolsRedactor struct {
	keys       map[string]struct{}
	headerKeys map[string]struct{}
}

func newDevToolsRedactor(keys []string, headers []string) devToolsRedactor {
	redactor := devToolsRedactor{
		keys:       map[string]struct{}{},
		headerKeys: map[string]struct{}{},
	}
	for _, key := range append(append([]string{}, defaultDevToolsRedactKeys...), keys...) {
		if normalized := strings.ToLower(strings.TrimSpace(key)); normalized != "" {
			redactor.keys[normalized] = struct{}{}
		}
	}
	for _, header := range append(append([]string{}, defaultDevToolsRedactHeaders...), headers...) {
		if normalized := strings.ToLower(strings.TrimSpace(header)); normalized != "" {
			redactor.headerKeys[normalized] = struct{}{}
		}
	}
	return redactor
}

func (r devToolsRedactor) headers(headers http.Header) map[string]string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	redacted := make(map[string]string, len(keys))
	for _, key := range keys {
		normalized := strings.ToLower(key)
		if _, secret := r.headerKeys[normalized]; secret {
			redacted[normalized] = devToolsRedactedValue
			continue
		}
		value := strings.Join(headers.Values(key), ", ")
		if normalized == "location" || normalized == strings.ToLower(HeaderInertiaLocation) || normalized == "referer" {
			value = r.url(value)
		}
		redacted[normalized] = value
	}
	return redacted
}

func (r devToolsRedactor) value(value any) any {
	switch current := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(current))
		for key, child := range current {
			if r.secretKey(key) {
				redacted[key] = devToolsRedactedValue
			} else if stringValue, ok := child.(string); ok && isDevToolsURLKey(key) {
				redacted[key] = r.url(stringValue)
			} else {
				redacted[key] = r.value(child)
			}
		}
		return redacted
	case []any:
		redacted := make([]any, len(current))
		for index, child := range current {
			redacted[index] = r.value(child)
		}
		return redacted
	default:
		return value
	}
}

func (r devToolsRedactor) secretKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if _, secret := r.keys[normalized]; secret {
		return true
	}
	normalized = strings.Trim(strings.NewReplacer("[", ".", "]", "").Replace(normalized), ".")
	if _, secret := r.keys[normalized]; secret {
		return true
	}
	for _, segment := range strings.Split(normalized, ".") {
		if _, secret := r.keys[segment]; secret {
			return true
		}
	}
	return false
}

func isDevToolsURLKey(key string) bool {
	return strings.EqualFold(key, "url") || strings.EqualFold(key, "redirectLocation")
}

func (r devToolsRedactor) url(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return value
	}
	if parsed.User != nil {
		parsed.User = url.User(devToolsRedactedValue)
	}
	query := parsed.Query()
	changed := false
	for key := range query {
		if r.secretQueryKey(key) {
			query.Set(key, devToolsRedactedValue)
			changed = true
		}
	}
	if changed {
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}

func (r devToolsRedactor) urlPointer(value *string) *string {
	if value == nil {
		return nil
	}
	return nonEmptyString(r.url(*value))
}

func (r devToolsRedactor) secretQueryKey(key string) bool {
	return r.secretKey(key)
}

type devToolsBodyReader struct {
	body      io.ReadCloser
	mu        sync.Mutex
	buffer    bytes.Buffer
	limit     int
	readBytes int64
	sawEOF    bool
	overflow  bool
}

func newDevToolsBodyReader(body io.ReadCloser, limit int) *devToolsBodyReader {
	return &devToolsBodyReader{body: body, limit: limit}
}

func (r *devToolsBodyReader) Read(target []byte) (int, error) {
	read, err := r.body.Read(target)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.readBytes += int64(read)
	if err == io.EOF {
		r.sawEOF = true
	}
	if read > 0 {
		remaining := r.limit - r.buffer.Len()
		if remaining > 0 {
			captured := min(read, remaining)
			_, _ = r.buffer.Write(target[:captured])
			if captured < read {
				r.overflow = true
			}
		} else {
			r.overflow = true
		}
	}
	return read, err
}

func (r *devToolsBodyReader) Close() error {
	return r.body.Close()
}

func (r *devToolsBodyReader) snapshot(contentLength int64) ([]byte, bool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	complete := r.sawEOF
	if contentLength >= 0 {
		complete = r.readBytes >= contentLength
	}
	return bytes.Clone(r.buffer.Bytes()), r.overflow, complete
}

func captureDevToolsRequestBody(req *http.Request, reader *devToolsBodyReader, maxBodyBytes int, redactor devToolsRedactor) devToolsBodyCapture {
	if requestMethodHasBody(req.Method) && !IsInertiaRequest(req) {
		return devToolsOmittedBody("non-inertia-request")
	}
	if req.ContentLength > int64(maxBodyBytes) {
		return devToolsOmittedBody("too-large")
	}

	var body []byte
	var overflow bool
	complete := req.ContentLength == 0
	if reader != nil {
		body, overflow, complete = reader.snapshot(req.ContentLength)
	}
	if overflow || len(body) > maxBodyBytes {
		return devToolsOmittedBody("too-large")
	}
	if !complete {
		return devToolsOmittedBody("streamed")
	}

	rawContentType := strings.TrimSpace(req.Header.Get("Content-Type"))
	contentType, parameters, contentTypeErr := mime.ParseMediaType(rawContentType)
	isMultipart := strings.HasPrefix(strings.ToLower(rawContentType), "multipart/") || strings.HasPrefix(contentType, "multipart/")
	if isMultipart {
		var value map[string]any
		if req.MultipartForm != nil {
			value = multipartDevToolsValue(req, redactor)
		} else if len(body) > 0 && contentTypeErr == nil && parameters["boundary"] != "" {
			parsed, err := multipartDevToolsValueFromBody(body, parameters["boundary"], redactor)
			if err != nil {
				return devToolsOmittedBody("non-textual")
			}
			value = parsed
		} else if len(body) > 0 {
			return devToolsOmittedBody("non-textual")
		}
		if len(value) == 0 {
			return devToolsEmptyBody()
		}
		return devToolsPresentBody(value)
	}
	if len(body) == 0 {
		return devToolsEmptyBody()
	}

	if strings.Contains(contentType, "json") {
		var value any
		if err := json.Unmarshal(body, &value); err == nil {
			return devToolsPresentBody(redactor.value(value))
		}
		return devToolsOmittedBody("unserializable")
	}
	if contentType == "application/x-www-form-urlencoded" {
		if values, err := url.ParseQuery(string(body)); err == nil {
			value := make(map[string]any, len(values))
			for key, entries := range values {
				if len(entries) == 1 {
					value[key] = entries[0]
				} else {
					value[key] = append([]string(nil), entries...)
				}
			}
			normalized, _ := normalizeDevToolsJSON(value)
			return devToolsPresentBody(redactor.value(normalized))
		}
		return devToolsOmittedBody("unserializable")
	}
	if contentTypeErr != nil && rawContentType != "" {
		return devToolsOmittedBody("non-textual")
	}
	if !isTextualDevToolsMediaType(contentType) {
		return devToolsOmittedBody("non-textual")
	}
	if !utf8.Valid(body) {
		return devToolsOmittedBody("binary")
	}
	return devToolsPresentBody(string(body))
}

func isTextualDevToolsMediaType(contentType string) bool {
	if contentType == "" || strings.HasPrefix(contentType, "text/") {
		return true
	}
	if strings.HasSuffix(contentType, "+json") || strings.HasSuffix(contentType, "+xml") {
		return true
	}
	switch contentType {
	case "application/json", "application/xml", "application/javascript", "application/graphql":
		return true
	default:
		return false
	}
}

func requestMethodHasBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func multipartDevToolsValue(req *http.Request, redactor devToolsRedactor) map[string]any {
	value := map[string]any{}
	for key, entries := range req.MultipartForm.Value {
		if len(entries) == 1 {
			value[key] = entries[0]
		} else {
			value[key] = append([]string(nil), entries...)
		}
	}
	for key, files := range req.MultipartForm.File {
		summaries := make([]any, 0, len(files))
		for _, file := range files {
			summaries = append(summaries, map[string]any{
				"name":     file.Filename,
				"size":     file.Size,
				"mimeType": file.Header.Get("Content-Type"),
			})
		}
		if len(summaries) == 1 {
			value[key] = summaries[0]
		} else {
			value[key] = summaries
		}
	}
	normalized, _ := normalizeDevToolsJSON(value)
	redacted, _ := redactor.value(normalized).(map[string]any)
	return redacted
}

func multipartDevToolsValueFromBody(body []byte, boundary string, redactor devToolsRedactor) (map[string]any, error) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	value := map[string]any{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		contents, readErr := io.ReadAll(part)
		_ = part.Close()
		if readErr != nil {
			return nil, readErr
		}
		name := part.FormName()
		if name == "" {
			continue
		}
		var captured any = string(contents)
		if filename := part.FileName(); filename != "" {
			captured = map[string]any{
				"name":     filename,
				"size":     len(contents),
				"mimeType": part.Header.Get("Content-Type"),
			}
		}
		appendDevToolsMultipartValue(value, name, captured)
	}
	normalized, err := normalizeDevToolsJSON(value)
	if err != nil {
		return nil, err
	}
	redacted, _ := redactor.value(normalized).(map[string]any)
	return redacted, nil
}

func appendDevToolsMultipartValue(values map[string]any, key string, value any) {
	current, exists := values[key]
	if !exists {
		values[key] = value
		return
	}
	if entries, ok := current.([]any); ok {
		values[key] = append(entries, value)
		return
	}
	values[key] = []any{current, value}
}

func normalizeDevToolsJSON(value any) (any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var normalized any
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

func (r *Renderer) recordDevToolsPage(req *http.Request, page Page, sources Props, shared map[string]bool) {
	if r.devTools == nil {
		return
	}
	state := devToolsState(req)
	if state == nil {
		return
	}
	defer func() {
		_ = recover()
	}()

	responseBody := devToolsOmittedBody("unserializable")
	normalizedPage, err := normalizeDevToolsJSON(page)
	if err == nil {
		responseBody = devToolsPresentBody(r.devTools.redactor.value(normalizedPage))
	}
	props := classifyDevToolsProps(req, sources, shared, page)
	propValues := map[string]any{}
	if normalizedProps, normalizeErr := normalizeDevToolsJSON(page.Props); normalizeErr == nil {
		if values, ok := r.devTools.redactor.value(normalizedProps).(map[string]any); ok {
			for path := range props {
				if value, exists := devToolsValueAtPath(values, path); exists {
					propValues[path] = value
				}
			}
		}
	}

	var componentPath *string
	if resolver := r.devTools.componentPathResolver; resolver != nil {
		componentPath = safeDevToolsComponentPath(resolver, page.Component)
	}
	state.setPage(page.Component, responseBody, props, propValues, componentPath)
}

func safeDevToolsComponentPath(resolver DevToolsComponentPathResolver, component string) (path *string) {
	defer func() {
		if recover() != nil {
			path = nil
		}
	}()
	return nonEmptyString(resolver(component))
}

func (r *Renderer) recordDevToolsRenderCaller(req *http.Request, skip int) {
	if r.devTools == nil {
		return
	}
	state := devToolsState(req)
	if state == nil {
		return
	}
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return
	}
	state.setRenderSource(devToolsSource{File: file, Line: line}, false)
}

func classifyDevToolsProps(req *http.Request, sources Props, shared map[string]bool, page Page) map[string]devToolsPropMeta {
	props := make(map[string]devToolsPropMeta, len(sources)+1)
	for _, path := range sortedPropKeys(sources) {
		sharedProp := shared[rootPropPath(path)]
		props[path] = classifyDevToolsProp(req, path, sources[path], sharedProp)
		classifyNestedDevToolsProps(req, path, sources[path], sharedProp, props)
	}
	props["errors"] = devToolsPropMeta{InertiaType: "always", Shared: true}
	for _, path := range page.MergeProps {
		meta := props[path]
		setDevToolsPropType(&meta, "merge")
		meta.MergeDirection = "append"
		props[path] = meta
	}
	for _, path := range page.PrependProps {
		meta := props[path]
		setDevToolsPropType(&meta, "merge")
		meta.MergeDirection = "prepend"
		props[path] = meta
	}
	for _, path := range page.DeepMergeProps {
		meta := props[path]
		setDevToolsPropType(&meta, "merge")
		if meta.MergeDirection == "" {
			meta.MergeDirection = "append"
		}
		meta.DeepMerge = true
		props[path] = meta
	}
	for _, once := range page.OnceProps {
		meta := props[once.Prop]
		setDevToolsPropType(&meta, "once")
		meta.Once = true
		props[once.Prop] = meta
	}
	for group, paths := range page.DeferredProps {
		for _, path := range paths {
			meta := props[path]
			setDevToolsPropType(&meta, "defer")
			meta.DeferGroup = group
			props[path] = meta
		}
	}
	for path, scroll := range page.ScrollProps {
		meta := props[path]
		setDevToolsPropType(&meta, "scroll")
		meta.MergeDirection = "append"
		if InfiniteScrollMergeIntent(req) == "prepend" {
			meta.MergeDirection = "prepend"
		}
		meta.Reset = meta.Reset || scroll.Reset
		props[path] = meta
	}
	for _, path := range ResetProps(req) {
		meta := props[path]
		meta.Reset = true
		props[path] = meta
	}
	for _, path := range page.RescuedProps {
		meta := props[path]
		meta.Rescued = true
		props[path] = meta
	}
	for path, meta := range props {
		if path == "errors" {
			continue
		}
		meta.Shared = shared[rootPropPath(path)]
		props[path] = meta
	}
	return props
}

func classifyNestedDevToolsProps(req *http.Request, path string, value any, shared bool, props map[string]devToolsPropMeta) {
	classifyNestedDevToolsPropsAtDepth(req, path, value, shared, props, map[devToolsReflectionVisit]struct{}{}, 0)
}

type devToolsReflectionVisit struct {
	typeOf  reflect.Type
	pointer uintptr
}

func classifyNestedDevToolsPropsAtDepth(
	req *http.Request,
	path string,
	value any,
	shared bool,
	props map[string]devToolsPropMeta,
	visiting map[devToolsReflectionVisit]struct{},
	depth int,
) {
	if depth >= devToolsMaximumPropDepth {
		return
	}
	if _, ok := devToolsPropValue(value); ok || value == nil || implementsJSONMarshaler(value) {
		return
	}

	marked := make([]devToolsReflectionVisit, 0, 2)
	defer func() {
		for _, visit := range marked {
			delete(visiting, visit)
		}
	}()
	mark := func(reflected reflect.Value) bool {
		visit := devToolsReflectionVisit{typeOf: reflected.Type(), pointer: reflected.Pointer()}
		if _, found := visiting[visit]; found {
			return false
		}
		visiting[visit] = struct{}{}
		marked = append(marked, visit)
		return true
	}

	reflected := reflect.ValueOf(value)
	for dereferences := 0; reflected.IsValid() && (reflected.Kind() == reflect.Interface || reflected.Kind() == reflect.Pointer); dereferences++ {
		if dereferences >= devToolsMaximumPropDepth {
			return
		}
		if reflected.IsNil() {
			return
		}
		if reflected.Kind() == reflect.Pointer && !mark(reflected) {
			return
		}
		reflected = reflected.Elem()
	}
	if !reflected.IsValid() {
		return
	}
	switch reflected.Kind() {
	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String || reflected.IsNil() {
			return
		}
		if !mark(reflected) {
			return
		}
		for _, key := range reflected.MapKeys() {
			childPath := joinPropPath(path, key.String())
			child := reflected.MapIndex(key).Interface()
			if _, ok := devToolsPropValue(child); ok {
				props[childPath] = classifyDevToolsProp(req, childPath, child, shared)
			}
			classifyNestedDevToolsPropsAtDepth(req, childPath, child, shared, props, visiting, depth+1)
		}
	case reflect.Slice:
		if reflected.Type().Elem().Kind() == reflect.Uint8 || reflected.IsNil() {
			return
		}
		if !mark(reflected) {
			return
		}
		fallthrough
	case reflect.Array:
		for index := 0; index < reflected.Len(); index++ {
			childPath := joinPropPath(path, strconv.Itoa(index))
			child := reflected.Index(index).Interface()
			if _, ok := devToolsPropValue(child); ok {
				props[childPath] = classifyDevToolsProp(req, childPath, child, shared)
			}
			classifyNestedDevToolsPropsAtDepth(req, childPath, child, shared, props, visiting, depth+1)
		}
	}
}

func setDevToolsPropType(meta *devToolsPropMeta, inertiaType string) {
	if devToolsPropTypePriority(inertiaType) > devToolsPropTypePriority(meta.InertiaType) {
		meta.InertiaType = inertiaType
	}
}

func devToolsPropTypePriority(inertiaType string) int {
	switch inertiaType {
	case "always":
		return 6
	case "scroll":
		return 5
	case "defer":
		return 4
	case "optional":
		return 3
	case "merge":
		return 2
	case "once":
		return 1
	default:
		return 0
	}
}

func devToolsValueAtPath(values map[string]any, path string) (any, bool) {
	var current any = values
	for _, segment := range strings.Split(path, ".") {
		switch value := current.(type) {
		case map[string]any:
			child, ok := value[segment]
			if !ok {
				return nil, false
			}
			current = child
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(value) {
				return nil, false
			}
			current = value[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func classifyDevToolsProp(req *http.Request, path string, value any, shared bool) devToolsPropMeta {
	meta := devToolsPropMeta{Shared: shared, Reset: containsString(ResetProps(req), path)}
	prop, ok := devToolsPropValue(value)
	if !ok {
		return meta
	}

	switch {
	case prop.mode == propModeAlways:
		meta.InertiaType = "always"
	case prop.scroll:
		meta.InertiaType = "scroll"
	case prop.deferred && hasDevToolsHeader(req, HeaderInertiaDevToolsDeferred):
		meta.InertiaType = "defer"
	case prop.mode == propModeOptional:
		meta.InertiaType = "optional"
	case prop.merge:
		meta.InertiaType = "merge"
	case prop.once:
		meta.InertiaType = "once"
	}
	if prop.deferred && (prop.scroll || hasDevToolsHeader(req, HeaderInertiaDevToolsDeferred)) {
		meta.DeferGroup = prop.group
	}
	meta.Once = prop.once
	if prop.merge || prop.scroll {
		meta.MergeDirection = "append"
		if prop.scroll {
			if InfiniteScrollMergeIntent(req) == "prepend" {
				meta.MergeDirection = "prepend"
			}
		} else if len(prop.prependPaths) > 0 && len(prop.appendPaths) == 0 {
			meta.MergeDirection = "prepend"
		}
	}
	meta.DeepMerge = prop.deepMerge || len(prop.matchOn) > 0
	return meta
}

func devToolsPropValue(value any) (Prop, bool) {
	switch prop := value.(type) {
	case Prop:
		return prop, true
	case *Prop:
		if prop != nil {
			return *prop, true
		}
	case PropFunc, func(*http.Request) (any, error):
		return newProp(value), true
	}
	return Prop{}, false
}

func devToolsIDScript(req *http.Request) []byte {
	state := devToolsState(req)
	if state == nil {
		return nil
	}
	encoded, err := json.Marshal(state.id)
	if err != nil {
		return nil
	}
	return append(append([]byte(`<script data-inertia-devtools-id type="application/json">`), encoded...), []byte(`</script>`)...)
}

func injectDevToolsIDScript(body []byte, script []byte) []byte {
	if len(script) == 0 {
		return body
	}
	index := bytes.LastIndex(bytes.ToLower(body), []byte("</body>"))
	if index < 0 {
		return append(append(bytes.Clone(body), script...), '\n')
	}
	result := make([]byte, 0, len(body)+len(script))
	result = append(result, body[:index]...)
	result = append(result, script...)
	result = append(result, body[index:]...)
	return result
}

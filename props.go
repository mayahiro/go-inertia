package inertia

import (
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// SharedPropsProvider returns props that are shared by every Inertia page.
type SharedPropsProvider interface {
	// Props returns shared props for req.
	Props(req *http.Request) (Props, error)
}

// SharedPropsFunc adapts a function to SharedPropsProvider.
type SharedPropsFunc func(req *http.Request) (Props, error)

// Props calls f(req).
func (f SharedPropsFunc) Props(req *http.Request) (Props, error) {
	return f(req)
}

// StaticSharedProps returns a provider that always returns props.
func StaticSharedProps(props Props) SharedPropsProvider {
	return SharedPropsFunc(func(req *http.Request) (Props, error) {
		return cloneProps(props), nil
	})
}

// NoSharedProps returns a provider that returns no shared props.
func NoSharedProps() SharedPropsProvider {
	return SharedPropsFunc(func(req *http.Request) (Props, error) {
		return Props{}, nil
	})
}

func (r *Renderer) page(req *http.Request, component string, props Props, opts renderOptions) (Page, error) {
	version, err := r.versionProvider.Version(req.Context())
	if err != nil {
		return Page{}, err
	}

	merged := newPageProps(component)

	shared, err := r.sharedProps.Props(req)
	if err != nil {
		return Page{}, err
	}

	combined := Props{}
	merged.mergeSharedSource(combined, shared)
	merged.mergeSharedSource(combined, SharedPropsFromContext(req.Context()))
	mergePropSource(combined, props)
	mergePropSource(combined, PropsFromContext(req.Context()))
	if opts.serverHead != nil {
		combined[r.serverHeadProp] = Always(opts.serverHead.Elements())
	}

	resolvedSources, err := unpackDotProps(req, combined)
	if err != nil {
		return Page{}, err
	}
	for _, key := range sortedPropKeys(resolvedSources) {
		if err := merged.set(req, key, resolvedSources[key]); err != nil {
			return Page{}, err
		}
	}

	flashData := FlashData{}
	if r.flashStore != nil {
		pulled, err := r.flashStore.Pull(req)
		if err != nil {
			return Page{}, err
		}
		flashData = pulled
	}

	pageFlash := Flash{}
	contextFlash := FlashFromContext(req.Context())
	if len(flashData.Flash) > 0 || len(contextFlash) > 0 {
		mergeFlash(pageFlash, flashData.Flash)
		mergeFlash(pageFlash, contextFlash)
	}

	errors := ValidationErrors{}
	mergeErrors(errors, flashData.Errors)
	mergeErrors(errors, ValidationErrorsFromContext(req.Context()))
	if bag := ErrorBag(req); bag != "" {
		if bagErrors, ok := flashData.Bags[bag]; ok {
			errors = ValidationErrors{bag: Props(bagErrors)}
		}
	}
	merged.Props["errors"] = Props(errors)

	merged.Metadata.filterForReset(req)

	page := Page{
		Component:        component,
		Props:            merged.Props,
		URL:              r.urlResolver.URL(req),
		Version:          version,
		EncryptHistory:   opts.encryptHistory,
		ClearHistory:     opts.clearHistory,
		PreserveFragment: opts.preserveFragment,
		SharedProps:      merged.sharedPropNames(),
		Flash:            pageFlash,
	}
	merged.Metadata.applyTo(&page)
	return page, nil
}

func (p *pageProps) mergeSharedSource(dst Props, src Props) {
	mergePropSource(dst, src)
	for key := range src {
		root := rootPropPath(key)
		if !isReservedProp(root) {
			p.SharedProps[root] = true
		}
	}
}

func mergePropSource(dst Props, src Props) {
	for key, value := range src {
		if isReservedProp(rootPropPath(key)) {
			continue
		}
		dst[key] = value
	}
}

func unpackDotProps(req *http.Request, src Props) (Props, error) {
	result := Props{}
	dotKeys := make([]string, 0)
	for key, value := range src {
		if strings.Contains(key, ".") {
			dotKeys = append(dotKeys, key)
			continue
		}
		result[key] = value
	}
	sort.Strings(dotKeys)
	for _, key := range dotKeys {
		value := src[key]
		if isPropFunc(value) {
			resolved, err := newProp(value).resolveValue(req)
			if err != nil {
				return nil, err
			}
			value = resolved
		}
		if err := setDotProp(req, result, strings.Split(key, "."), value); err != nil {
			return nil, err
		}
	}
	for key, value := range result {
		result[key] = normalizeDotValue(value)
	}
	return result, nil
}

func setDotProp(req *http.Request, props Props, segments []string, value any) error {
	if len(segments) == 0 || segments[0] == "" {
		return nil
	}
	if len(segments) == 1 {
		props[segments[0]] = value
		return nil
	}
	nested, err := dotPropsFromValue(req, props[segments[0]])
	if err != nil {
		return err
	}
	if err := setDotProp(req, nested, segments[1:], value); err != nil {
		return err
	}
	props[segments[0]] = nested
	return nil
}

func dotPropsFromValue(req *http.Request, value any) (Props, error) {
	if isPropFunc(value) {
		resolved, err := newProp(value).resolveValue(req)
		if err != nil {
			return nil, err
		}
		value = resolved
	}
	if value == nil {
		return Props{}, nil
	}

	reflected := reflect.ValueOf(value)
	for reflected.IsValid() && (reflected.Kind() == reflect.Interface || reflected.Kind() == reflect.Pointer) {
		if reflected.IsNil() {
			return Props{}, nil
		}
		reflected = reflected.Elem()
	}
	if !reflected.IsValid() {
		return Props{}, nil
	}

	props := Props{}
	switch reflected.Kind() {
	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String || reflected.IsNil() {
			return props, nil
		}
		for _, key := range reflected.MapKeys() {
			props[key.String()] = reflected.MapIndex(key).Interface()
		}
	case reflect.Slice:
		if reflected.Type().Elem().Kind() == reflect.Uint8 || reflected.IsNil() {
			return props, nil
		}
		fallthrough
	case reflect.Array:
		for index := 0; index < reflected.Len(); index++ {
			props[strconv.Itoa(index)] = reflected.Index(index).Interface()
		}
	}
	return props, nil
}

func normalizeDotValue(value any) any {
	if value == nil || implementsJSONMarshaler(value) {
		return value
	}
	reflected := reflect.ValueOf(value)
	for reflected.IsValid() && (reflected.Kind() == reflect.Interface || reflected.Kind() == reflect.Pointer) {
		if reflected.IsNil() {
			return value
		}
		reflected = reflected.Elem()
	}
	if !reflected.IsValid() {
		return value
	}

	switch reflected.Kind() {
	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String || reflected.IsNil() {
			return value
		}
		props := Props{}
		for _, key := range reflected.MapKeys() {
			props[key.String()] = normalizeDotValue(reflected.MapIndex(key).Interface())
		}
		if sequence, ok := numericPropsSequence(props); ok {
			return sequence
		}
		return props
	case reflect.Slice:
		if reflected.Type().Elem().Kind() == reflect.Uint8 || reflected.IsNil() {
			return value
		}
		fallthrough
	case reflect.Array:
		sequence := make([]any, reflected.Len())
		for index := 0; index < reflected.Len(); index++ {
			sequence[index] = normalizeDotValue(reflected.Index(index).Interface())
		}
		return sequence
	default:
		return value
	}
}

func numericPropsSequence(props Props) ([]any, bool) {
	if len(props) == 0 {
		return nil, false
	}
	sequence := make([]any, len(props))
	for key, value := range props {
		index, err := strconv.Atoi(key)
		if err != nil || index < 0 || index >= len(props) || strconv.Itoa(index) != key {
			return nil, false
		}
		sequence[index] = value
	}
	return sequence, true
}

func sortedPropKeys(props Props) []string {
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func rootPropPath(path string) string {
	if index := strings.IndexByte(path, '.'); index >= 0 {
		return path[:index]
	}
	return path
}

func isReservedProp(key string) bool {
	return key == "errors" || key == "flash"
}

func (p pageProps) sharedPropNames() []string {
	if len(p.SharedProps) == 0 {
		return nil
	}
	names := make([]string, 0, len(p.SharedProps))
	for key := range p.SharedProps {
		names = append(names, key)
	}
	sort.Strings(names)
	return names
}

func mergeErrors(dst ValidationErrors, src ValidationErrors) {
	for key, value := range src {
		dst[key] = value
	}
}

func mergeFlash(dst Flash, src Flash) {
	for key, value := range src {
		dst[key] = value
	}
}

func cloneProps(src Props) Props {
	dst := Props{}
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func isPartialReloadForComponent(req *http.Request, component string) bool {
	return IsInertiaRequest(req) && PartialComponent(req) == component
}

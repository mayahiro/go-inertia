package inertia

import (
	"errors"
	"net/http"
	"time"
)

var errInvalidPropFunc = errors.New("inertia: prop function is nil")

// PropFunc loads a prop for req.
type PropFunc func(req *http.Request) (any, error)

// Prop is a page prop with optional Inertia protocol modifiers.
type Prop struct {
	value any
	mode  propMode

	deferred bool
	group    string
	rescue   bool

	merge        bool
	appendPaths  []string
	prependPaths []string
	deepMerge    bool
	matchOn      []string

	once      bool
	onceKey   string
	expiresAt *int64
	fresh     bool

	scroll         bool
	scrollMetadata ScrollMetadata
	scrollWrapper  string
}

type propMode int

const (
	propModeValue propMode = iota
	propModeLazy
	propModeOptional
	propModeAlways
)

// Lazy returns a prop that is evaluated only when the response includes it.
func Lazy(fn PropFunc) Prop {
	return newProp(fn)
}

// Optional returns a prop that is only included when explicitly requested.
func Optional(value any) Prop {
	return newProp(value).Optional()
}

// Always returns a prop that is included even during partial reloads.
func Always(value any) Prop {
	return newProp(value).Always()
}

func newProp(value any) Prop {
	p := Prop{
		value:         value,
		group:         "default",
		scrollWrapper: "data",
	}
	if isPropFunc(value) {
		p.mode = propModeLazy
	}
	return p
}

func isPropFunc(value any) bool {
	switch value.(type) {
	case PropFunc, func(*http.Request) (any, error):
		return true
	default:
		return false
	}
}

// Optional returns p configured to be included only when explicitly requested.
func (p Prop) Optional() Prop {
	p.mode = propModeOptional
	return p
}

// Always returns p configured to be included even during partial reloads.
func (p Prop) Always() Prop {
	p.mode = propModeAlways
	return p
}

// Defer returns p configured as a deferred prop.
func (p Prop) Defer(group ...string) Prop {
	p.deferred = true
	p.group = deferredGroup(group)
	return p
}

// Rescue omits a failed deferred prop and marks it in Page.RescuedProps.
func (p Prop) Rescue(rescue ...bool) Prop {
	p.rescue = true
	if len(rescue) > 0 {
		p.rescue = rescue[0]
	}
	return p
}

// Merge returns p configured to append at the root path.
func (p Prop) Merge() Prop {
	p.merge = true
	return p
}

// Append returns p configured to append the given relative prop paths.
func (p Prop) Append(paths ...string) Prop {
	p.merge = true
	p.appendPaths = append(p.appendPaths, normalizeMergePaths(paths)...)
	return p
}

// Prepend returns p configured to prepend the given relative prop paths.
func (p Prop) Prepend(paths ...string) Prop {
	p.merge = true
	p.prependPaths = append(p.prependPaths, normalizeMergePaths(paths)...)
	return p
}

// DeepMerge returns p configured to deeply merge the whole prop.
func (p Prop) DeepMerge() Prop {
	p.merge = true
	p.deepMerge = true
	return p
}

// MatchOn returns p configured to match merged items by the given relative paths.
func (p Prop) MatchOn(paths ...string) Prop {
	p.matchOn = append(p.matchOn, paths...)
	return p
}

// Once returns p configured to be reused by the client after it is loaded.
func (p Prop) Once() Prop {
	p.once = true
	return p
}

// As returns p with a custom once key shared across matching once props.
func (p Prop) As(key string) Prop {
	p.once = true
	p.onceKey = key
	return p
}

// Fresh returns p configured to ignore the client's remembered once value.
func (p Prop) Fresh(fresh ...bool) Prop {
	p.once = true
	p.fresh = true
	if len(fresh) > 0 {
		p.fresh = fresh[0]
	}
	return p
}

// Until returns p with an expiration timestamp sent to the client.
func (p Prop) Until(t time.Time) Prop {
	p.once = true
	expiresAt := t.UnixMilli()
	p.expiresAt = &expiresAt
	return p
}

// Wrapper returns p configured to merge a custom infinite scroll data wrapper.
func (p Prop) Wrapper(path string) Prop {
	p.scrollWrapper = path
	return p
}

func (p Prop) resolveProp(req *http.Request, component string, key string) (propResult, error) {
	if p.deferred {
		return p.resolveDeferred(req, component, key)
	}
	if p.usesRememberedOnce(req, component, key) {
		return propResult{Omit: true, Metadata: p.onceMetadata(key)}, nil
	}
	if !p.includes(req, component, key) {
		return propResult{Omit: true}, nil
	}
	return p.resolveIncluded(req, key)
}

func (p Prop) resolveDeferred(req *http.Request, component string, key string) (propResult, error) {
	if p.usesRememberedOnce(req, component, key) {
		return propResult{Omit: true, Metadata: p.onceMetadata(key)}, nil
	}
	if !isPartialReloadForComponent(req, component) {
		return propResult{
			Omit:     true,
			Metadata: deferredPropMetadata(p.group, key),
		}, nil
	}
	if !partialReloadIncludesProp(req, key) {
		return propResult{Omit: true}, nil
	}
	result, err := p.resolveIncluded(req, key)
	if err != nil && p.rescue {
		return propResult{
			Omit:     true,
			Metadata: rescuedPropMetadata(key),
		}, nil
	}
	return result, err
}

func (p Prop) resolveIncluded(req *http.Request, key string) (propResult, error) {
	value, err := p.resolveValue(req)
	if err != nil {
		return propResult{}, err
	}
	return propResult{
		Value:    value,
		Metadata: p.metadata(req, key),
		Always:   p.mode == propModeAlways,
	}, nil
}

func (p Prop) resolveValue(req *http.Request) (any, error) {
	switch value := p.value.(type) {
	case PropFunc:
		if value == nil {
			return nil, errInvalidPropFunc
		}
		return value(req)
	case func(*http.Request) (any, error):
		if value == nil {
			return nil, errInvalidPropFunc
		}
		return value(req)
	default:
		return value, nil
	}
}

func (p Prop) includes(req *http.Request, component string, key string) bool {
	switch p.mode {
	case propModeOptional:
		return isPartialReloadForComponent(req, component) && containsString(PartialData(req), key)
	case propModeAlways:
		return true
	case propModeLazy:
		return !isPartialReloadForComponent(req, component) || partialReloadIncludesProp(req, key)
	default:
		return true
	}
}

func (p Prop) usesRememberedOnce(req *http.Request, component string, key string) bool {
	return p.once &&
		IsInertiaRequest(req) &&
		!isPartialReloadForComponent(req, component) &&
		!p.fresh &&
		containsString(ExceptOnceProps(req), p.resolvedOnceKey(key))
}

func (p Prop) metadata(req *http.Request, key string) pageMetadata {
	metadata := pageMetadata{}
	if p.merge {
		metadata.merge(p.mergeMetadata(key))
	}
	if p.scroll {
		metadata.merge(p.scrollPageMetadata(req, key))
	}
	if p.once {
		metadata.merge(p.onceMetadata(key))
	}
	return metadata
}

func (p Prop) mergeMetadata(key string) pageMetadata {
	metadata := pageMetadata{
		MergeProps:   prefixPropPaths(key, p.appendPaths),
		PrependProps: prefixPropPaths(key, p.prependPaths),
		MatchPropsOn: prefixPropPaths(key, p.matchOn),
	}
	if len(p.appendPaths) == 0 && len(p.prependPaths) == 0 && !p.deepMerge {
		metadata.MergeProps = []string{key}
	}
	if p.deepMerge {
		metadata.DeepMergeProps = []string{key}
	}
	return metadata
}

func (p Prop) scrollPageMetadata(req *http.Request, key string) pageMetadata {
	metadata := p.scrollMetadata
	if metadata.PageName == "" {
		metadata.PageName = "page"
	}
	if containsString(ResetProps(req), key) {
		metadata.Reset = true
	}

	path := scrollMergePath(key, p.scrollWrapper)
	pageMetadata := pageMetadata{
		MatchPropsOn: prefixPropPaths(key, p.matchOn),
		ScrollProps: map[string]ScrollMetadata{
			key: metadata,
		},
	}
	if InfiniteScrollMergeIntent(req) == "prepend" {
		pageMetadata.PrependProps = []string{path}
	} else {
		pageMetadata.MergeProps = []string{path}
	}
	return pageMetadata
}

func (p Prop) onceMetadata(key string) pageMetadata {
	onceKey := p.resolvedOnceKey(key)
	return pageMetadata{
		OnceProps: map[string]OncePropMetadata{
			onceKey: {
				Prop:      key,
				ExpiresAt: p.expiresAt,
			},
		},
	}
}

func (p Prop) resolvedOnceKey(prop string) string {
	if p.onceKey == "" {
		return prop
	}
	return p.onceKey
}

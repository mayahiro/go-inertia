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
	wrapperSet     bool

	invalidReason string
}

type propMode int

const (
	propModeValue propMode = iota
	propModeLazy
	propModeOptional
	propModeAlways
)

// Computed returns a normal prop whose callback is evaluated only when the response includes it.
func Computed(fn PropFunc) Prop {
	return newProp(fn)
}

// Lazy returns a prop that is evaluated only when the response includes it.
//
// Deprecated: use Computed. Lazy is not the removed Inertia v3 LazyProp type.
func Lazy(fn PropFunc) Prop {
	return Computed(fn)
}

// Optional returns a prop omitted from full visits and selected by the only and
// except filters of a partial reload for the same component. An except-only
// reload includes the prop unless it is excluded.
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

// Optional returns p configured to be omitted from full visits and selected by
// the only and except filters of a partial reload for the same component.
// An except-only reload includes the prop unless it is excluded.
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
	if len(group) > 1 {
		p.invalidReason = "Defer accepts at most one group"
	}
	p.deferred = true
	p.group = deferredGroup(group)
	return p
}

// Rescue omits a failed deferred prop and marks it in Page.RescuedProps.
func (p Prop) Rescue(rescue ...bool) Prop {
	if len(rescue) > 1 {
		p.invalidReason = "Rescue accepts at most one boolean"
	}
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
	if len(fresh) > 1 {
		p.invalidReason = "Fresh accepts at most one boolean"
	}
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
	p.wrapperSet = true
	return p
}

// Scroll returns p configured as an infinite scroll prop.
func (p Prop) Scroll(metadata ScrollMetadata) Prop {
	p.scroll = true
	p.scrollMetadata = metadata
	return p
}

func (p Prop) resolveProp(context propResolutionContext) (propResult, error) {
	if err := p.validate(context.Path); err != nil {
		return propResult{}, err
	}
	if p.deferred {
		return p.resolveDeferred(context)
	}
	if p.mode == propModeOptional && !isPartialReloadForComponent(context.Request, context.Component) {
		return propResult{Omit: true, Metadata: p.initialOmittedMetadata(context)}, nil
	}
	if p.usesRememberedOnce(context) {
		return propResult{Omit: true, Metadata: p.onceMetadata(context.Path)}, nil
	}
	if !p.includes(context) {
		return propResult{Omit: true}, nil
	}
	return p.resolveIncluded(context)
}

func (p Prop) resolveDeferred(context propResolutionContext) (propResult, error) {
	if !isPartialReloadForComponent(context.Request, context.Component) {
		return propResult{
			Omit:     true,
			Metadata: p.initialOmittedMetadata(context),
		}, nil
	}
	if !partialPathIncluded(context) {
		return propResult{Omit: true}, nil
	}
	result, err := p.resolveIncluded(context)
	if err != nil && p.rescue {
		return propResult{
			Omit:     true,
			Metadata: rescuedPropMetadata(context.Path),
		}, nil
	}
	return result, err
}

func (p Prop) initialOmittedMetadata(context propResolutionContext) pageMetadata {
	metadata := pageMetadata{}
	if p.deferred && !p.usesRememberedOnce(context) {
		metadata.merge(deferredPropMetadata(p.group, context.Path))
	}
	if p.merge {
		metadata.merge(p.mergeMetadata(context.Path))
	}
	if p.scroll {
		scrollMetadata := p.scrollPageMetadata(context.Request, context.Path)
		scrollMetadata.ScrollProps = nil
		metadata.merge(scrollMetadata)
	}
	if p.once {
		metadata.merge(p.onceMetadata(context.Path))
	}
	return metadata
}

func (p Prop) resolveIncluded(context propResolutionContext) (propResult, error) {
	value, err := p.resolveValue(context.Request)
	if err != nil {
		return propResult{}, err
	}
	return propResult{
		Value:    value,
		Metadata: p.metadata(context),
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

func (p Prop) includes(context propResolutionContext) bool {
	if context.ParentWasResolved && isPartialReloadForComponent(context.Request, context.Component) {
		return true
	}
	switch p.mode {
	case propModeOptional:
		return isPartialReloadForComponent(context.Request, context.Component) && partialPathIncluded(context)
	case propModeAlways:
		return true
	case propModeLazy:
		return !isPartialReloadForComponent(context.Request, context.Component) || partialPathIncluded(context)
	default:
		return true
	}
}

func (p Prop) usesRememberedOnce(context propResolutionContext) bool {
	return p.once &&
		IsInertiaRequest(context.Request) &&
		!isPartialReloadForComponent(context.Request, context.Component) &&
		!p.fresh &&
		containsString(ExceptOnceProps(context.Request), p.resolvedOnceKey(context.Path))
}

func (p Prop) metadata(context propResolutionContext) pageMetadata {
	metadata := pageMetadata{}
	if p.merge && partialMetadataIncludesProp(context.Request, context.Component, context.Path) {
		metadata.merge(p.mergeMetadata(context.Path))
	}
	if p.scroll {
		metadata.merge(p.scrollPageMetadata(context.Request, context.Path))
	}
	if p.once && partialMetadataIncludesProp(context.Request, context.Component, context.Path) {
		metadata.merge(p.onceMetadata(context.Path))
	}
	return metadata
}

func (p Prop) validate(path string) error {
	reason := p.invalidReason
	if reason == "" && p.rescue && !p.deferred {
		reason = "Rescue requires Defer"
	}
	if reason == "" && len(p.matchOn) > 0 && !p.merge && !p.scroll {
		reason = "MatchOn requires Merge or Scroll"
	}
	if reason == "" && p.wrapperSet && !p.scroll {
		reason = "Wrapper requires Scroll"
	}
	if reason == "" {
		return nil
	}
	return &PropConfigurationError{Path: path, Reason: reason}
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

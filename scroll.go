package inertia

import "net/http"

// ScrollFunc loads an infinite scroll prop for req.
type ScrollFunc func(req *http.Request) (any, error)

// ScrollMetadata describes pagination state for an infinite scroll prop.
type ScrollMetadata struct {
	// PageName is the query parameter used for this scroll container.
	PageName string `json:"pageName"`
	// PreviousPage is the previous page value, or nil when there is no previous page.
	PreviousPage any `json:"previousPage"`
	// NextPage is the next page value, or nil when there is no next page.
	NextPage any `json:"nextPage"`
	// CurrentPage is the current page value.
	CurrentPage any `json:"currentPage"`
	// Reset marks the scroll prop as reset for the current response.
	Reset bool `json:"reset,omitempty"`
}

// ScrollProp marks a page prop as an infinite scroll prop.
type ScrollProp struct {
	value    any
	metadata ScrollMetadata
	wrapper  string
	matchOn  []string
}

// Scroll returns an infinite scroll prop with explicit pagination metadata.
func Scroll(value any, metadata ScrollMetadata) ScrollProp {
	return ScrollProp{
		value:    value,
		metadata: metadata,
		wrapper:  "data",
	}
}

// Wrapper returns p configured to merge a custom data wrapper path.
func (p ScrollProp) Wrapper(path string) ScrollProp {
	p.wrapper = path
	return p
}

// MatchOn returns p configured to match merged items by the given relative paths.
func (p ScrollProp) MatchOn(paths ...string) ScrollProp {
	p.matchOn = append(p.matchOn, paths...)
	return p
}

func (p ScrollProp) resolveProp(req *http.Request, component string, key string) (propResult, error) {
	value, err := p.resolveValue(req)
	if err != nil {
		return propResult{}, err
	}
	return propResult{
		Value:    value,
		Metadata: p.pageMetadata(req, key),
	}, nil
}

func (p ScrollProp) resolveValue(req *http.Request) (any, error) {
	switch value := p.value.(type) {
	case ScrollFunc:
		return value(req)
	case func(*http.Request) (any, error):
		return value(req)
	default:
		return value, nil
	}
}

func (p ScrollProp) pageMetadata(req *http.Request, key string) pageMetadata {
	metadata := p.metadata
	if metadata.PageName == "" {
		metadata.PageName = "page"
	}
	if containsString(ResetProps(req), key) {
		metadata.Reset = true
	}

	path := scrollMergePath(key, p.wrapper)
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

func scrollMergePath(key string, wrapper string) string {
	if wrapper == "" {
		return key
	}
	return key + "." + wrapper
}

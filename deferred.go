package inertia

import (
	"errors"
	"net/http"
)

var errInvalidDeferredProp = errors.New("inertia: deferred prop requires a function")

// DeferredFunc loads a deferred prop for req.
type DeferredFunc func(req *http.Request) (any, error)

// DeferredProp marks a page prop as deferred until a matching partial reload.
type DeferredProp struct {
	fn    DeferredFunc
	group string
}

// Defer returns a prop that is omitted from the initial page response.
func Defer(fn DeferredFunc, group ...string) DeferredProp {
	return DeferredProp{
		fn:    fn,
		group: deferredGroup(group),
	}
}

func (p DeferredProp) resolveProp(req *http.Request, component string, key string) (propResult, error) {
	if !isPartialReloadForComponent(req, component) {
		return propResult{
			Omit:     true,
			Metadata: deferredPropMetadata(p.group, key),
		}, nil
	}

	if !partialReloadIncludesProp(req, key) {
		return propResult{Omit: true}, nil
	}

	if p.fn == nil {
		return propResult{}, errInvalidDeferredProp
	}

	value, err := p.fn(req)
	if err != nil {
		return propResult{}, err
	}
	return propResult{Value: value}, nil
}

func deferredGroup(groups []string) string {
	if len(groups) == 0 || groups[0] == "" {
		return "default"
	}
	return groups[0]
}

func deferredPropMetadata(group string, key string) pageMetadata {
	return pageMetadata{
		DeferredProps: map[string][]string{
			group: {key},
		},
	}
}

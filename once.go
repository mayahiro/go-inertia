package inertia

import (
	"errors"
	"net/http"
	"time"
)

var errInvalidOnceProp = errors.New("inertia: once prop requires a function")

// OnceFunc loads a prop that the Inertia client remembers across visits.
type OnceFunc func(req *http.Request) (any, error)

// OnceProp marks a page prop as reusable after the client receives it once.
type OnceProp struct {
	fn        OnceFunc
	key       string
	expiresAt *int64
	fresh     bool
}

// Once returns a prop that is resolved once and then reused by the client.
func Once(fn OnceFunc) OnceProp {
	return OnceProp{fn: fn}
}

// As returns p with a custom once key shared across matching once props.
func (p OnceProp) As(key string) OnceProp {
	p.key = key
	return p
}

// Fresh returns p configured to ignore the client's remembered once value.
func (p OnceProp) Fresh(fresh ...bool) OnceProp {
	p.fresh = true
	if len(fresh) > 0 {
		p.fresh = fresh[0]
	}
	return p
}

// Until returns p with an expiration timestamp sent to the client.
func (p OnceProp) Until(t time.Time) OnceProp {
	expiresAt := t.UnixMilli()
	p.expiresAt = &expiresAt
	return p
}

func (p OnceProp) resolveProp(req *http.Request, component string, key string) (propResult, error) {
	onceKey := p.onceKey(key)
	metadata := oncePropMetadata(onceKey, key, p.expiresAt)

	if isPartialReloadForComponent(req, component) {
		if !partialReloadIncludesProp(req, key) {
			return propResult{Omit: true}, nil
		}
		return p.resolve(req, metadata)
	}

	if IsInertiaRequest(req) && !p.fresh && containsString(ExceptOnceProps(req), onceKey) {
		return propResult{Omit: true, Metadata: metadata}, nil
	}

	return p.resolve(req, metadata)
}

func (p OnceProp) resolve(req *http.Request, metadata pageMetadata) (propResult, error) {
	if p.fn == nil {
		return propResult{}, errInvalidOnceProp
	}

	value, err := p.fn(req)
	if err != nil {
		return propResult{}, err
	}
	return propResult{Value: value, Metadata: metadata}, nil
}

func (p OnceProp) onceKey(prop string) string {
	if p.key == "" {
		return prop
	}
	return p.key
}

func oncePropMetadata(key string, prop string, expiresAt *int64) pageMetadata {
	return pageMetadata{
		OnceProps: map[string]OncePropMetadata{
			key: {
				Prop:      prop,
				ExpiresAt: expiresAt,
			},
		},
	}
}

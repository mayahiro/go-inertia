package inertia

import "net/http"

// MergeFunc loads a mergeable prop for req.
type MergeFunc func(req *http.Request) (any, error)

// MergeProp marks a page prop as mergeable during partial reloads.
type MergeProp struct {
	value        any
	appendPaths  []string
	prependPaths []string
	deepMerge    bool
	matchOn      []string
}

// Merge returns a prop that appends at the root path by default.
func Merge(value any) MergeProp {
	return MergeProp{value: value}
}

// Append returns p configured to append the given relative prop paths.
func (p MergeProp) Append(paths ...string) MergeProp {
	p.appendPaths = append(p.appendPaths, normalizeMergePaths(paths)...)
	return p
}

// Prepend returns p configured to prepend the given relative prop paths.
func (p MergeProp) Prepend(paths ...string) MergeProp {
	p.prependPaths = append(p.prependPaths, normalizeMergePaths(paths)...)
	return p
}

// DeepMerge returns p configured to deeply merge the whole prop.
func (p MergeProp) DeepMerge() MergeProp {
	p.deepMerge = true
	return p
}

// MatchOn returns p configured to match merged items by the given relative paths.
func (p MergeProp) MatchOn(paths ...string) MergeProp {
	p.matchOn = append(p.matchOn, paths...)
	return p
}

func (p MergeProp) resolveProp(req *http.Request, component string, key string) (propResult, error) {
	value, err := p.resolveValue(req)
	if err != nil {
		return propResult{}, err
	}
	return propResult{
		Value:    value,
		Metadata: p.metadata(key),
	}, nil
}

func (p MergeProp) resolveValue(req *http.Request) (any, error) {
	switch value := p.value.(type) {
	case MergeFunc:
		return value(req)
	case func(*http.Request) (any, error):
		return value(req)
	default:
		return value, nil
	}
}

func (p MergeProp) metadata(key string) pageMetadata {
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

func normalizeMergePaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{""}
	}
	return paths
}

func prefixPropPaths(key string, paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	prefixed := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			prefixed = append(prefixed, key)
		} else {
			prefixed = append(prefixed, key+"."+path)
		}
	}
	return prefixed
}

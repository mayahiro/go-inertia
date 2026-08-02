package inertia

import (
	"net/http"
	"strings"
)

type propResult struct {
	Value    any
	Metadata pageMetadata
	Omit     bool
}

type propResolver interface {
	resolveProp(context propResolutionContext) (propResult, error)
}

type propResolutionContext struct {
	Request           *http.Request
	Component         string
	Path              string
	ParentWasResolved bool
}

type pageProps struct {
	Props       Props
	Metadata    pageMetadata
	SharedProps map[string]bool
	Component   string
}

type pageMetadata struct {
	MergeProps     []string
	PrependProps   []string
	DeepMergeProps []string
	MatchPropsOn   []string
	ScrollProps    map[string]ScrollMetadata
	DeferredProps  map[string][]string
	RescuedProps   []string
	OnceProps      map[string]OncePropMetadata
}

func newPageProps(component string) pageProps {
	return pageProps{
		Props:       Props{},
		SharedProps: map[string]bool{},
		Component:   component,
	}
}

func (p *pageProps) set(req *http.Request, key string, value any) error {
	if err := validateStaticPropTree(key, value); err != nil {
		return err
	}
	p.Metadata.remove(key)

	result, err := resolvePropTree(propResolutionContext{
		Request:   req,
		Component: p.Component,
		Path:      key,
	}, value)
	if err != nil {
		return err
	}

	if result.Omit {
		delete(p.Props, key)
	} else {
		p.Props[key] = result.Value
	}
	p.Metadata.merge(result.Metadata)
	return nil
}

func resolveProp(context propResolutionContext, value any) (propResult, error) {
	if prop, ok := value.(*Prop); ok && prop == nil {
		return propResult{}, &PropConfigurationError{Path: context.Path, Reason: "prop must not be nil"}
	}
	if isPropFunc(value) {
		return newProp(value).resolveProp(context)
	}
	resolver, ok := value.(propResolver)
	if !ok {
		return propResult{Value: value}, nil
	}
	return resolver.resolveProp(context)
}

func (m *pageMetadata) remove(key string) {
	m.MergeProps = filterPropPaths(m.MergeProps, key)
	m.PrependProps = filterPropPaths(m.PrependProps, key)
	m.DeepMergeProps = filterPropPaths(m.DeepMergeProps, key)
	m.MatchPropsOn = filterPropPaths(m.MatchPropsOn, key)

	delete(m.ScrollProps, key)

	for group, props := range m.DeferredProps {
		props = filterExactProp(props, key)
		if len(props) == 0 {
			delete(m.DeferredProps, group)
		} else {
			m.DeferredProps[group] = props
		}
	}

	m.RescuedProps = filterExactProp(m.RescuedProps, key)

	for onceKey, once := range m.OnceProps {
		if onceKey == key || once.Prop == key {
			delete(m.OnceProps, onceKey)
		}
	}
}

func (m *pageMetadata) merge(other pageMetadata) {
	m.MergeProps = appendUnique(m.MergeProps, other.MergeProps...)
	m.PrependProps = appendUnique(m.PrependProps, other.PrependProps...)
	m.DeepMergeProps = appendUnique(m.DeepMergeProps, other.DeepMergeProps...)
	m.MatchPropsOn = appendUnique(m.MatchPropsOn, other.MatchPropsOn...)

	if len(other.ScrollProps) > 0 {
		if m.ScrollProps == nil {
			m.ScrollProps = map[string]ScrollMetadata{}
		}
		for key, value := range other.ScrollProps {
			m.ScrollProps[key] = value
		}
	}

	if len(other.DeferredProps) > 0 {
		if m.DeferredProps == nil {
			m.DeferredProps = map[string][]string{}
		}
		for group, props := range other.DeferredProps {
			m.DeferredProps[group] = appendUnique(m.DeferredProps[group], props...)
		}
	}

	m.RescuedProps = appendUnique(m.RescuedProps, other.RescuedProps...)

	if len(other.OnceProps) > 0 {
		if m.OnceProps == nil {
			m.OnceProps = map[string]OncePropMetadata{}
		}
		for key, value := range other.OnceProps {
			m.OnceProps[key] = value
		}
	}
}

func (m pageMetadata) applyTo(page *Page) {
	page.MergeProps = m.MergeProps
	page.PrependProps = m.PrependProps
	page.DeepMergeProps = m.DeepMergeProps
	page.MatchPropsOn = m.MatchPropsOn
	page.ScrollProps = m.ScrollProps
	page.DeferredProps = m.DeferredProps
	page.RescuedProps = m.RescuedProps
	page.OnceProps = m.OnceProps
}

func (m *pageMetadata) filterForReset(req *http.Request) {
	for _, key := range ResetProps(req) {
		m.removeMerge(key)
		if scroll, ok := m.ScrollProps[key]; ok {
			scroll.Reset = true
			m.ScrollProps[key] = scroll
		}
	}
}

func (m *pageMetadata) removeMerge(key string) {
	m.MergeProps = filterPropPaths(m.MergeProps, key)
	m.PrependProps = filterPropPaths(m.PrependProps, key)
	m.DeepMergeProps = filterPropPaths(m.DeepMergeProps, key)
	m.MatchPropsOn = filterPropPaths(m.MatchPropsOn, key)

	if len(m.ScrollProps) == 0 {
		m.ScrollProps = nil
	}
}

func filterPropPaths(paths []string, key string) []string {
	if len(paths) == 0 {
		return nil
	}
	filtered := paths[:0]
	for _, path := range paths {
		if path != key && !strings.HasPrefix(path, key+".") {
			filtered = append(filtered, path)
		}
	}
	return filtered
}

func filterExactProp(props []string, key string) []string {
	if len(props) == 0 {
		return nil
	}
	filtered := props[:0]
	for _, prop := range props {
		if prop != key {
			filtered = append(filtered, prop)
		}
	}
	return filtered
}

func appendUnique(dst []string, values ...string) []string {
	for _, value := range values {
		if !containsString(dst, value) {
			dst = append(dst, value)
		}
	}
	return dst
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

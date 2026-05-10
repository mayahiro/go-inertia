package inertia

import (
	"net/http"
	"strings"
)

type propResult struct {
	Value    any
	Metadata pageMetadata
	Omit     bool
	Always   bool
}

type propResolver interface {
	resolveProp(req *http.Request, component string, key string) (propResult, error)
}

type pageProps struct {
	Props       Props
	Metadata    pageMetadata
	AlwaysProps map[string]bool
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
		AlwaysProps: map[string]bool{},
		SharedProps: map[string]bool{},
		Component:   component,
	}
}

func (p *pageProps) mergePublicProps(req *http.Request, src Props) error {
	for key, value := range src {
		if isReservedProp(key) {
			continue
		}
		if err := p.set(req, key, value); err != nil {
			return err
		}
		delete(p.SharedProps, key)
	}
	return nil
}

func (p *pageProps) set(req *http.Request, key string, value any) error {
	p.Metadata.remove(key)
	delete(p.AlwaysProps, key)

	result, err := resolveProp(req, p.Component, key, value)
	if err != nil {
		return err
	}

	if result.Omit {
		delete(p.Props, key)
	} else {
		p.Props[key] = result.Value
	}
	if result.Always {
		p.AlwaysProps[key] = true
	}
	p.Metadata.merge(result.Metadata)
	return nil
}

func resolveProp(req *http.Request, component string, key string, value any) (propResult, error) {
	if isPropFunc(value) {
		return newProp(value).resolveProp(req, component, key)
	}
	resolver, ok := value.(propResolver)
	if !ok {
		return propResult{Value: value}, nil
	}
	return resolver.resolveProp(req, component, key)
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

func (m *pageMetadata) filterForProps(props Props) {
	m.MergeProps = filterExistingPropPaths(m.MergeProps, props)
	m.PrependProps = filterExistingPropPaths(m.PrependProps, props)
	m.DeepMergeProps = filterExistingPropPaths(m.DeepMergeProps, props)
	m.MatchPropsOn = filterExistingPropPaths(m.MatchPropsOn, props)

	for key := range m.ScrollProps {
		if _, ok := props[key]; !ok {
			delete(m.ScrollProps, key)
		}
	}
	if len(m.ScrollProps) == 0 {
		m.ScrollProps = nil
	}
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

func filterExistingPropPaths(paths []string, props Props) []string {
	if len(paths) == 0 {
		return nil
	}
	filtered := paths[:0]
	for _, path := range paths {
		if propPathExists(props, path) {
			filtered = append(filtered, path)
		}
	}
	return filtered
}

func propPathExists(props Props, path string) bool {
	if index := strings.IndexByte(path, '.'); index >= 0 {
		path = path[:index]
	}
	_, ok := props[path]
	return ok
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

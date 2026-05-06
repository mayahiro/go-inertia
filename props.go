package inertia

import "net/http"

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
	if err := merged.mergePublicProps(req, shared); err != nil {
		return Page{}, err
	}
	if err := merged.mergePublicProps(req, SharedPropsFromContext(req.Context())); err != nil {
		return Page{}, err
	}
	if err := merged.mergePublicProps(req, props); err != nil {
		return Page{}, err
	}
	if err := merged.mergePublicProps(req, PropsFromContext(req.Context())); err != nil {
		return Page{}, err
	}

	flashData := FlashData{}
	if r.flashStore != nil {
		pulled, err := r.flashStore.Pull(req)
		if err != nil {
			return Page{}, err
		}
		flashData = pulled
	}

	contextFlash := FlashFromContext(req.Context())
	if len(flashData.Flash) > 0 || len(contextFlash) > 0 {
		flash := Props{}
		mergePublicProps(flash, Props(flashData.Flash))
		mergePublicProps(flash, Props(contextFlash))
		if len(flash) > 0 {
			merged.Props["flash"] = flash
		}
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

	pageProps := applyPartialReload(req, component, merged.Props)
	merged.Metadata.filterForProps(pageProps)

	page := Page{
		Component:        component,
		Props:            pageProps,
		URL:              r.urlResolver.URL(req),
		Version:          version,
		PreserveFragment: opts.preserveFragment,
	}
	merged.Metadata.applyTo(&page)
	return page, nil
}

func mergePublicProps(dst Props, src Props) {
	for key, value := range src {
		if key == "errors" || key == "flash" {
			continue
		}
		dst[key] = value
	}
}

func mergeErrors(dst ValidationErrors, src ValidationErrors) {
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

func applyPartialReload(req *http.Request, component string, props Props) Props {
	if !isPartialReloadForComponent(req, component) {
		return props
	}

	if except := PartialExcept(req); len(except) > 0 {
		filtered := cloneProps(props)
		for _, key := range except {
			if key != "errors" && key != "flash" {
				delete(filtered, key)
			}
		}
		if _, ok := filtered["errors"]; !ok {
			filtered["errors"] = Props{}
		}
		return filtered
	}

	if data := PartialData(req); len(data) > 0 {
		filtered := Props{}
		for _, key := range data {
			if value, ok := props[key]; ok {
				filtered[key] = value
			}
		}
		filtered["errors"] = props["errors"]
		if flash, ok := props["flash"]; ok {
			filtered["flash"] = flash
		}
		return filtered
	}

	return props
}

func isPartialReloadForComponent(req *http.Request, component string) bool {
	return IsInertiaRequest(req) && PartialComponent(req) == component
}

func partialReloadIncludesProp(req *http.Request, key string) bool {
	if except := PartialExcept(req); len(except) > 0 {
		return !containsString(except, key)
	}
	if data := PartialData(req); len(data) > 0 {
		return containsString(data, key)
	}
	return true
}

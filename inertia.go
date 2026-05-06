package inertia

import (
	"errors"
	"net/http"
)

var (
	// ErrMissingRootView is returned when Config.RootView is not set.
	ErrMissingRootView = errors.New("inertia: missing root view")
	// ErrMissingFlashStore is returned when flash data is used without a FlashStore.
	ErrMissingFlashStore = errors.New("inertia: flash store is not configured")
	// ErrInvalidComponent is returned when a render call receives an empty component name.
	ErrInvalidComponent = errors.New("inertia: component must not be empty")
)

// Renderer renders Inertia pages, handles protocol middleware, and creates Inertia redirects.
type Renderer struct {
	rootView        RootView
	versionProvider VersionProvider
	sharedProps     SharedPropsProvider
	flashStore      FlashStore
	urlResolver     URLResolver
	jsonEncoder     JSONEncoder
	renderOptions   []RenderOption
}

// Config configures a Renderer.
type Config struct {
	// RootView renders the initial HTML document.
	RootView RootView
	// VersionProvider returns the current asset version.
	VersionProvider VersionProvider
	// SharedProps returns props that are merged into every page.
	SharedProps SharedPropsProvider
	// FlashStore stores one-time flash data and validation errors across redirects.
	FlashStore FlashStore
	// URLResolver returns the URL written into the Inertia page object.
	URLResolver URLResolver
	// JSONEncoder encodes Inertia page objects.
	JSONEncoder          JSONEncoder
	DefaultRenderOptions []RenderOption
}

// New creates a Renderer from config.
func New(config Config) (*Renderer, error) {
	if config.RootView == nil {
		return nil, ErrMissingRootView
	}

	versionProvider := config.VersionProvider
	if versionProvider == nil {
		versionProvider = StaticVersion("")
	}

	sharedProps := config.SharedProps
	if sharedProps == nil {
		sharedProps = NoSharedProps()
	}

	urlResolver := config.URLResolver
	if urlResolver == nil {
		urlResolver = URLResolverFunc(func(req *http.Request) string {
			if req.URL == nil {
				return ""
			}
			return req.URL.RequestURI()
		})
	}

	jsonEncoder := config.JSONEncoder
	if jsonEncoder == nil {
		jsonEncoder = StandardJSONEncoder{}
	}

	return &Renderer{
		rootView:        config.RootView,
		versionProvider: versionProvider,
		sharedProps:     sharedProps,
		flashStore:      config.FlashStore,
		urlResolver:     urlResolver,
		jsonEncoder:     jsonEncoder,
		renderOptions:   append([]RenderOption(nil), config.DefaultRenderOptions...),
	}, nil
}

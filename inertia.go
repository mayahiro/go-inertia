package inertia

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrMissingRootView is returned when Config.RootView is not set.
	ErrMissingRootView = errors.New("inertia: missing root view")
	// ErrMissingFlashStore is returned when flash data is used without a FlashStore.
	ErrMissingFlashStore = errors.New("inertia: flash store is not configured")
	// ErrInvalidComponent is returned when a render call receives an empty component name.
	ErrInvalidComponent = errors.New("inertia: component must not be empty")
	// ErrComponentNotFound is returned when a configured component checker cannot find a component.
	ErrComponentNotFound = errors.New("inertia: component not found")
	// ErrInvalidScrollPaginator is returned when ScrollPage receives a nil paginator.
	ErrInvalidScrollPaginator = errors.New("inertia: scroll paginator is nil")
	// ErrInvalidPropConfiguration is returned for a modifier combination that has no valid protocol meaning.
	ErrInvalidPropConfiguration = errors.New("inertia: invalid prop configuration")
	// ErrInvalidRootElementID is returned when Config.RootElementID is not safe for the Inertia DOM selectors.
	ErrInvalidRootElementID = errors.New("inertia: invalid root element id")
	// ErrInvalidServerHeadProp is returned when Config.ServerHeadProp is not a valid top-level prop name.
	ErrInvalidServerHeadProp = errors.New("inertia: invalid server head prop")
	// ErrInvalidHeadElement is returned when a server head element cannot be generated safely.
	ErrInvalidHeadElement = errors.New("inertia: invalid head element")
	// ErrMissingFlashSessionResolver is returned when SessionFlashStore has no session resolver.
	ErrMissingFlashSessionResolver = errors.New("inertia: missing flash session resolver")
	// ErrMissingFlashSession is returned when a resolver does not return a session.
	ErrMissingFlashSession = errors.New("inertia: missing flash session")
)

// PropConfigurationError describes an invalid prop modifier combination.
type PropConfigurationError struct {
	// Path is the dot-notation path of the invalid prop.
	Path string
	// Reason explains why the modifier combination is invalid.
	Reason string
}

// Error returns the invalid prop configuration message.
func (e *PropConfigurationError) Error() string {
	return fmt.Sprintf("%s: %s: %s", ErrInvalidPropConfiguration, e.Path, e.Reason)
}

// Unwrap makes PropConfigurationError match ErrInvalidPropConfiguration.
func (e *PropConfigurationError) Unwrap() error {
	return ErrInvalidPropConfiguration
}

// Renderer renders Inertia pages, handles protocol middleware, and creates Inertia redirects.
type Renderer struct {
	rootView             RootView
	versionProvider      VersionProvider
	sharedProps          SharedPropsProvider
	flashStore           FlashStore
	urlResolver          URLResolver
	jsonEncoder          JSONEncoder
	renderOptions        []RenderOption
	componentTransformer ComponentNameTransformer
	componentChecker     ComponentExistenceChecker
	rootElementID        string
	serverHeadProp       string
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
	JSONEncoder JSONEncoder
	// DefaultRenderOptions are applied to every Render call before request options.
	DefaultRenderOptions []RenderOption
	// ComponentNameTransformer transforms component names before rendering.
	ComponentNameTransformer ComponentNameTransformer
	// ComponentExistenceChecker checks transformed component names before rendering.
	ComponentExistenceChecker ComponentExistenceChecker
	// RootElementID is the DOM id and data-page value used to mount the client application.
	// The default is "app".
	RootElementID string
	// ServerHeadProp is the page prop read by the client serverHead option.
	// The default is "head".
	ServerHeadProp string
}

// New creates a Renderer from config.
func New(config Config) (*Renderer, error) {
	if config.RootView == nil {
		return nil, ErrMissingRootView
	}

	rootElementID := config.RootElementID
	if rootElementID == "" {
		rootElementID = "app"
	}
	if !validRootElementID(rootElementID) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidRootElementID, rootElementID)
	}

	serverHeadProp := config.ServerHeadProp
	if serverHeadProp == "" {
		serverHeadProp = "head"
	}
	if !validServerHeadProp(serverHeadProp) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidServerHeadProp, serverHeadProp)
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
		rootView:             config.RootView,
		versionProvider:      versionProvider,
		sharedProps:          sharedProps,
		flashStore:           config.FlashStore,
		urlResolver:          urlResolver,
		jsonEncoder:          jsonEncoder,
		renderOptions:        append([]RenderOption(nil), config.DefaultRenderOptions...),
		componentTransformer: config.ComponentNameTransformer,
		componentChecker:     config.ComponentExistenceChecker,
		rootElementID:        rootElementID,
		serverHeadProp:       serverHeadProp,
	}, nil
}

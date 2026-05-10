package inertia

// Props is the set of top-level props sent to the Inertia page component.
type Props map[string]any

// Page is the Inertia page object sent to the browser.
type Page struct {
	// Component is the client-side component name.
	Component string `json:"component"`
	// Props contains the data for the client-side component.
	Props Props `json:"props"`
	// URL is the current request URL as seen by Inertia.
	URL string `json:"url"`
	// Version is the current asset version.
	Version any `json:"version,omitempty"`
	// EncryptHistory requests encrypted browser history state when supported by the client.
	EncryptHistory bool `json:"encryptHistory,omitempty"`
	// ClearHistory requests clearing browser history state when supported by the client.
	ClearHistory bool `json:"clearHistory,omitempty"`
	// PreserveFragment requests preserving the current URL fragment when supported by the client.
	PreserveFragment bool `json:"preserveFragment,omitempty"`
	// RescuedProps lists deferred props that failed and were rescued.
	RescuedProps []string `json:"rescuedProps,omitempty"`
	// SharedProps lists shared prop names when supported by the client.
	SharedProps []string `json:"sharedProps,omitempty"`
	// MergeProps lists prop paths that should be appended during navigation.
	MergeProps []string `json:"mergeProps,omitempty"`
	// PrependProps lists prop paths that should be prepended during navigation.
	PrependProps []string `json:"prependProps,omitempty"`
	// DeepMergeProps lists prop paths that should be deeply merged during navigation.
	DeepMergeProps []string `json:"deepMergeProps,omitempty"`
	// MatchPropsOn lists prop paths used to match items while merging arrays.
	MatchPropsOn []string `json:"matchPropsOn,omitempty"`
	// ScrollProps contains infinite scroll metadata keyed by prop name.
	ScrollProps map[string]ScrollMetadata `json:"scrollProps,omitempty"`
	// DeferredProps groups deferred prop names by request group.
	DeferredProps map[string][]string `json:"deferredProps,omitempty"`
	// OnceProps contains once prop metadata keyed by once prop key.
	OnceProps map[string]OncePropMetadata `json:"onceProps,omitempty"`
}

// OncePropMetadata describes an Inertia once prop entry in the page object.
type OncePropMetadata struct {
	// Prop is the page prop path reused by the client.
	Prop string `json:"prop"`
	// ExpiresAt is a Unix millisecond timestamp, or nil when the prop does not expire.
	ExpiresAt *int64 `json:"expiresAt"`
}

// ValidationErrors is a map of validation error values keyed by field or bag name.
type ValidationErrors map[string]any

// Flash is the set of one-time props sent after a redirect.
type Flash Props

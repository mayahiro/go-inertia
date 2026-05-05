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
	// SharedProps lists shared prop names when supported by the client.
	SharedProps []string `json:"sharedProps,omitempty"`
}

// ValidationErrors is a map of validation error values keyed by field or bag name.
type ValidationErrors map[string]any

// Flash is the set of one-time props sent after a redirect.
type Flash Props

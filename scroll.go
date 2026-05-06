package inertia

// ScrollFunc loads an infinite scroll prop for req.
type ScrollFunc = PropFunc

// ScrollMetadata describes pagination state for an infinite scroll prop.
type ScrollMetadata struct {
	// PageName is the query parameter used for this scroll container.
	PageName string `json:"pageName"`
	// PreviousPage is the previous page value, or nil when there is no previous page.
	PreviousPage any `json:"previousPage"`
	// NextPage is the next page value, or nil when there is no next page.
	NextPage any `json:"nextPage"`
	// CurrentPage is the current page value.
	CurrentPage any `json:"currentPage"`
	// Reset marks the scroll prop as reset for the current response.
	Reset bool `json:"reset,omitempty"`
}

// ScrollProp marks a page prop as an infinite scroll prop.
type ScrollProp = Prop

// Scroll returns an infinite scroll prop with explicit pagination metadata.
func Scroll(value any, metadata ScrollMetadata) ScrollProp {
	p := newProp(value)
	p.scroll = true
	p.scrollMetadata = metadata
	return p
}

func scrollMergePath(key string, wrapper string) string {
	if wrapper == "" {
		return key
	}
	return key + "." + wrapper
}

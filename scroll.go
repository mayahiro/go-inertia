package inertia

import (
	"net/http"
	"reflect"
)

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

// ScrollPaginator exposes paginated data in the shape required by ScrollPage.
type ScrollPaginator interface {
	// ScrollItems returns the items for the current page.
	ScrollItems() any
	// ScrollMetadata returns the pagination state for the current page.
	ScrollMetadata() ScrollMetadata
}

// Scroll returns an infinite scroll prop with explicit pagination metadata.
func Scroll(value any, metadata ScrollMetadata) ScrollProp {
	return newProp(value).Scroll(metadata)
}

// ScrollPage returns an infinite scroll prop from a paginator interface.
func ScrollPage(paginator ScrollPaginator, wrapper ...string) ScrollProp {
	path := "data"
	if len(wrapper) > 0 && wrapper[0] != "" {
		path = wrapper[0]
	}
	if isNilScrollPaginator(paginator) {
		return Scroll(func(_ *http.Request) (any, error) {
			return nil, ErrInvalidScrollPaginator
		}, ScrollMetadata{}).Wrapper(path)
	}
	return Scroll(Props{
		path: paginator.ScrollItems(),
	}, paginator.ScrollMetadata()).Wrapper(path)
}

func isNilScrollPaginator(paginator ScrollPaginator) bool {
	if paginator == nil {
		return true
	}
	value := reflect.ValueOf(paginator)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func scrollMergePath(key string, wrapper string) string {
	if wrapper == "" {
		return key
	}
	return key + "." + wrapper
}

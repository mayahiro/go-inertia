package inertia

// DeferredFunc loads a deferred prop for req.
type DeferredFunc = PropFunc

// DeferredProp marks a page prop as deferred until a matching partial reload.
type DeferredProp = Prop

// Defer returns a prop that is omitted from the initial page response.
func Defer(fn DeferredFunc, group ...string) DeferredProp {
	return newProp(fn).Defer(group...)
}

func deferredGroup(groups []string) string {
	if len(groups) == 0 || groups[0] == "" {
		return "default"
	}
	return groups[0]
}

func deferredPropMetadata(group string, key string) pageMetadata {
	return pageMetadata{
		DeferredProps: map[string][]string{
			group: {key},
		},
	}
}

func rescuedPropMetadata(key string) pageMetadata {
	return pageMetadata{
		RescuedProps: []string{key},
	}
}

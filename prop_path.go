package inertia

import (
	"net/http"
	"strings"
)

func partialPathIncluded(context propResolutionContext) bool {
	if context.ParentWasResolved || !isPartialReloadForComponent(context.Request, context.Component) {
		return true
	}
	if only := PartialData(context.Request); len(only) > 0 {
		if !propPathMatchesOnly(context.Path, only) && !propPathLeadsToOnly(context.Path, only) {
			return false
		}
	}
	if except := PartialExcept(context.Request); len(except) > 0 && propPathMatchesExcept(context.Path, except) {
		return false
	}
	return true
}

func partialMetadataIncludesProp(req *http.Request, component string, path string) bool {
	if !isPartialReloadForComponent(req, component) {
		return true
	}
	if only := PartialData(req); len(only) > 0 && !propPathMatchesOnly(path, only) {
		return false
	}
	if except := PartialExcept(req); len(except) > 0 && propPathMatchesExcept(path, except) {
		return false
	}
	return true
}

func propPathMatchesOnly(path string, only []string) bool {
	for _, candidate := range only {
		if path == candidate || strings.HasPrefix(path, candidate+".") {
			return true
		}
	}
	return false
}

func propPathLeadsToOnly(path string, only []string) bool {
	for _, candidate := range only {
		if strings.HasPrefix(candidate, path+".") {
			return true
		}
	}
	return false
}

func propPathMatchesExcept(path string, except []string) bool {
	for _, candidate := range except {
		if path == candidate || strings.HasPrefix(path, candidate+".") {
			return true
		}
	}
	return false
}

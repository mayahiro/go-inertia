package inertia

// MergeFunc loads a mergeable prop for req.
type MergeFunc = PropFunc

// MergeProp marks a page prop as mergeable during partial reloads.
type MergeProp = Prop

// Merge returns a prop that appends at the root path by default.
func Merge(value any) MergeProp {
	return newProp(value).Merge()
}

func normalizeMergePaths(paths []string) []string {
	if len(paths) == 0 {
		return []string{""}
	}
	return paths
}

func prefixPropPaths(key string, paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	prefixed := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			prefixed = append(prefixed, key)
		} else {
			prefixed = append(prefixed, key+"."+path)
		}
	}
	return prefixed
}

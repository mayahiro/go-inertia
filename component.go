package inertia

// ComponentNameTransformer transforms component names before rendering.
type ComponentNameTransformer func(component string) string

// ComponentExistenceChecker checks whether a component exists.
type ComponentExistenceChecker interface {
	// ComponentExists reports whether component exists.
	ComponentExists(component string) (bool, error)
}

// ComponentExistsFunc adapts a function to ComponentExistenceChecker.
type ComponentExistsFunc func(component string) (bool, error)

// ComponentExists calls f(component).
func (f ComponentExistsFunc) ComponentExists(component string) (bool, error) {
	return f(component)
}

// ComponentNotFoundError is returned when a configured checker cannot find a component.
type ComponentNotFoundError struct {
	// Component is the transformed component name that was checked.
	Component string
}

// Error returns the error message.
func (e ComponentNotFoundError) Error() string {
	return "inertia: component not found: " + e.Component
}

// Is reports whether target is ErrComponentNotFound.
func (e ComponentNotFoundError) Is(target error) bool {
	return target == ErrComponentNotFound
}

func (r *Renderer) prepareComponent(component string) (string, error) {
	if component == "" {
		return "", ErrInvalidComponent
	}
	if r.componentTransformer != nil {
		component = r.componentTransformer(component)
	}
	if component == "" {
		return "", ErrInvalidComponent
	}
	if r.componentChecker == nil {
		return component, nil
	}
	exists, err := r.componentChecker.ComponentExists(component)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ComponentNotFoundError{Component: component}
	}
	return component, nil
}

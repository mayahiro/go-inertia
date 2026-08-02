package inertia

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
)

func resolvePropTree(context propResolutionContext, value any) (propResult, error) {
	if !isAlwaysPropValue(value) && !partialPathIncluded(context) {
		return propResult{Omit: true}, nil
	}

	originalWasContainer := isPropContainerValue(value)
	result, err := resolveProp(context, value)
	if err != nil {
		return propResult{}, err
	}
	if result.Omit {
		return result, nil
	}

	if isResolvablePropValue(result.Value) {
		unwrapped, err := resolveProp(context, result.Value)
		if err != nil {
			return propResult{}, err
		}
		result.Metadata.merge(unwrapped.Metadata)
		result.Value = unwrapped.Value
		result.Omit = unwrapped.Omit
		if result.Omit {
			return result, nil
		}
	}

	parentWasResolved := context.ParentWasResolved || !originalWasContainer
	resolved, metadata, handled, err := resolvePropContainer(context, result.Value, parentWasResolved)
	if err != nil {
		return propResult{}, err
	}
	if handled {
		result.Value = resolved
		result.Metadata.merge(metadata)
	}
	return result, nil
}

func validateStaticPropTree(path string, value any) error {
	switch prop := value.(type) {
	case Prop:
		if err := prop.validate(path); err != nil {
			return err
		}
		return validateStaticPropTree(path, prop.value)
	case *Prop:
		if prop == nil {
			return &PropConfigurationError{Path: path, Reason: "prop must not be nil"}
		}
		if err := prop.validate(path); err != nil {
			return err
		}
		return validateStaticPropTree(path, prop.value)
	}
	if value == nil || isPropFunc(value) || implementsJSONMarshaler(value) {
		return nil
	}
	reflected := reflect.ValueOf(value)
	for reflected.IsValid() && (reflected.Kind() == reflect.Interface || reflected.Kind() == reflect.Pointer) {
		if reflected.IsNil() {
			return nil
		}
		reflected = reflected.Elem()
	}
	if !reflected.IsValid() {
		return nil
	}
	switch reflected.Kind() {
	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String || reflected.IsNil() {
			return nil
		}
		for _, key := range reflected.MapKeys() {
			if err := validateStaticPropTree(joinPropPath(path, key.String()), reflected.MapIndex(key).Interface()); err != nil {
				return err
			}
		}
	case reflect.Slice:
		if reflected.Type().Elem().Kind() == reflect.Uint8 || reflected.IsNil() {
			return nil
		}
		fallthrough
	case reflect.Array:
		for index := 0; index < reflected.Len(); index++ {
			if err := validateStaticPropTree(joinPropPath(path, strconv.Itoa(index)), reflected.Index(index).Interface()); err != nil {
				return err
			}
		}
	}
	return nil
}

func isResolvablePropValue(value any) bool {
	if isPropFunc(value) {
		return true
	}
	_, ok := value.(propResolver)
	return ok
}

func isAlwaysPropValue(value any) bool {
	switch prop := value.(type) {
	case Prop:
		return prop.mode == propModeAlways
	case *Prop:
		return prop != nil && prop.mode == propModeAlways
	default:
		return false
	}
}

func isPropContainerValue(value any) bool {
	if value == nil || implementsJSONMarshaler(value) {
		return false
	}
	reflected := reflect.ValueOf(value)
	for reflected.IsValid() && (reflected.Kind() == reflect.Interface || reflected.Kind() == reflect.Pointer) {
		if reflected.IsNil() {
			return false
		}
		reflected = reflected.Elem()
	}
	if !reflected.IsValid() {
		return false
	}
	switch reflected.Kind() {
	case reflect.Map:
		return reflected.Type().Key().Kind() == reflect.String
	case reflect.Slice:
		return reflected.Type().Elem().Kind() != reflect.Uint8
	case reflect.Array:
		return true
	default:
		return false
	}
}

func resolvePropContainer(context propResolutionContext, value any, parentWasResolved bool) (any, pageMetadata, bool, error) {
	if value == nil || implementsJSONMarshaler(value) {
		return value, pageMetadata{}, false, nil
	}

	reflected := reflect.ValueOf(value)
	for reflected.IsValid() && (reflected.Kind() == reflect.Interface || reflected.Kind() == reflect.Pointer) {
		if reflected.IsNil() {
			return value, pageMetadata{}, false, nil
		}
		reflected = reflected.Elem()
	}
	if !reflected.IsValid() {
		return value, pageMetadata{}, false, nil
	}

	switch reflected.Kind() {
	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String {
			return value, pageMetadata{}, false, nil
		}
		if reflected.IsNil() {
			return value, pageMetadata{}, true, nil
		}
		keys := reflected.MapKeys()
		sort.Slice(keys, func(i int, j int) bool {
			return keys[i].String() < keys[j].String()
		})
		resolved := Props{}
		metadata := pageMetadata{}
		for _, key := range keys {
			path := joinPropPath(context.Path, key.String())
			child, err := resolvePropTree(propResolutionContext{
				Request:           context.Request,
				Component:         context.Component,
				Path:              path,
				ParentWasResolved: parentWasResolved,
			}, reflected.MapIndex(key).Interface())
			if err != nil {
				return nil, pageMetadata{}, true, err
			}
			metadata.merge(child.Metadata)
			if !child.Omit {
				resolved[key.String()] = child.Value
			}
		}
		return resolved, metadata, true, nil
	case reflect.Slice:
		if reflected.Type().Elem().Kind() == reflect.Uint8 {
			return value, pageMetadata{}, false, nil
		}
		if reflected.IsNil() {
			return value, pageMetadata{}, true, nil
		}
		return resolvePropSequence(context, reflected, parentWasResolved)
	case reflect.Array:
		return resolvePropSequence(context, reflected, parentWasResolved)
	default:
		return value, pageMetadata{}, false, nil
	}
}

func resolvePropSequence(context propResolutionContext, value reflect.Value, parentWasResolved bool) (any, pageMetadata, bool, error) {
	resolved := make([]any, 0, value.Len())
	var sparse Props
	metadata := pageMetadata{}
	for index := 0; index < value.Len(); index++ {
		child, err := resolvePropTree(propResolutionContext{
			Request:           context.Request,
			Component:         context.Component,
			Path:              joinPropPath(context.Path, strconv.Itoa(index)),
			ParentWasResolved: parentWasResolved,
		}, value.Index(index).Interface())
		if err != nil {
			return nil, pageMetadata{}, true, err
		}
		metadata.merge(child.Metadata)
		if child.Omit {
			continue
		}
		if sparse == nil && index == len(resolved) {
			resolved = append(resolved, child.Value)
			continue
		}
		if sparse == nil {
			sparse = make(Props, len(resolved)+1)
			for resolvedIndex, resolvedValue := range resolved {
				sparse[strconv.Itoa(resolvedIndex)] = resolvedValue
			}
		}
		sparse[strconv.Itoa(index)] = child.Value
	}
	if sparse != nil {
		return sparse, metadata, true, nil
	}
	return resolved, metadata, true, nil
}

func implementsJSONMarshaler(value any) bool {
	_, ok := value.(json.Marshaler)
	return ok
}

func joinPropPath(parent string, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

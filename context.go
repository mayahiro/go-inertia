package inertia

import "context"

type propsContextKey struct{}
type sharedPropsContextKey struct{}
type validationErrorsContextKey struct{}
type flashContextKey struct{}

// WithProps stores request-scoped props in ctx.
func WithProps(ctx context.Context, props Props) context.Context {
	return context.WithValue(ctx, propsContextKey{}, props)
}

// PropsFromContext returns request-scoped props from ctx.
func PropsFromContext(ctx context.Context) Props {
	props, _ := ctx.Value(propsContextKey{}).(Props)
	if props == nil {
		return Props{}
	}
	return props
}

// WithSharedProps stores request-scoped shared props in ctx.
func WithSharedProps(ctx context.Context, props Props) context.Context {
	return context.WithValue(ctx, sharedPropsContextKey{}, props)
}

// SharedPropsFromContext returns request-scoped shared props from ctx.
func SharedPropsFromContext(ctx context.Context) Props {
	props, _ := ctx.Value(sharedPropsContextKey{}).(Props)
	if props == nil {
		return Props{}
	}
	return props
}

// WithValidationErrorsContext stores validation errors in ctx for the current request.
func WithValidationErrorsContext(ctx context.Context, errors ValidationErrors) context.Context {
	return context.WithValue(ctx, validationErrorsContextKey{}, errors)
}

// ValidationErrorsFromContext returns validation errors stored in ctx.
func ValidationErrorsFromContext(ctx context.Context) ValidationErrors {
	errors, _ := ctx.Value(validationErrorsContextKey{}).(ValidationErrors)
	if errors == nil {
		return ValidationErrors{}
	}
	return errors
}

// WithFlashContext stores flash data in ctx for the current request.
func WithFlashContext(ctx context.Context, flash Flash) context.Context {
	return context.WithValue(ctx, flashContextKey{}, flash)
}

// FlashFromContext returns flash data stored in ctx.
func FlashFromContext(ctx context.Context) Flash {
	flash, _ := ctx.Value(flashContextKey{}).(Flash)
	if flash == nil {
		return Flash{}
	}
	return flash
}

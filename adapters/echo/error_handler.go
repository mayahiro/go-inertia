package inertiaecho

import (
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	echo "github.com/labstack/echo/v5"
	inertia "github.com/mayahiro/go-inertia"
)

// ErrorPropsFunc returns page props for an Echo error response.
type ErrorPropsFunc func(c *echo.Context, err error, status int) (inertia.Props, error)

// ErrorRenderPredicate decides whether an Echo error should render an Inertia page.
type ErrorRenderPredicate func(c *echo.Context, err error, status int) bool

// ErrorRenderOptionsFunc returns render options for an Echo error response.
type ErrorRenderOptionsFunc func(c *echo.Context, err error, status int) ([]inertia.RenderOption, error)

// ErrorHandlerConfig configures Adapter.ErrorHandler.
type ErrorHandlerConfig struct {
	// Pages maps HTTP status codes to Inertia component names.
	Pages map[int]string
	// DefaultComponent is used when Pages has no exact status mapping.
	DefaultComponent string
	// Props optionally builds page props from the original error.
	Props ErrorPropsFunc
	// RenderOptions optionally builds render options such as WithServerHead.
	RenderOptions ErrorRenderOptionsFunc
	// ShouldRender optionally limits which requests receive Inertia error pages.
	// If nil, DefaultErrorRenderPredicate is used.
	ShouldRender ErrorRenderPredicate
	// Fallback handles requests that are not mapped or cannot be rendered.
	// The default is echo.DefaultHTTPErrorHandler(false).
	Fallback echo.HTTPErrorHandler
}

// ErrorHandler returns an Echo HTTP error handler that renders configured Inertia pages.
func (a *Adapter) ErrorHandler(config ErrorHandlerConfig) echo.HTTPErrorHandler {
	pages := make(map[int]string, len(config.Pages))
	for status, component := range config.Pages {
		pages[status] = component
	}
	fallback := config.Fallback
	if fallback == nil {
		fallback = echo.DefaultHTTPErrorHandler(false)
	}
	shouldRender := config.ShouldRender
	if shouldRender == nil {
		shouldRender = DefaultErrorRenderPredicate
	}

	return func(c *echo.Context, err error) {
		if responseCommitted(c) {
			return
		}
		status := echo.StatusCode(err)
		if status == 0 {
			status = http.StatusInternalServerError
		}
		component := pages[status]
		if component == "" {
			component = config.DefaultComponent
		}
		if component == "" {
			fallback(c, err)
			return
		}
		inertia.AppendVary(c.Response().Header(), echo.HeaderAccept)
		inertia.AppendVary(c.Response().Header(), inertia.HeaderInertia)
		if !shouldRender(c, err, status) {
			fallback(c, err)
			return
		}

		props := inertia.Props{}
		if config.Props != nil {
			resolved, propsErr := config.Props(c, err, status)
			if propsErr != nil {
				fallback(c, errors.Join(err, propsErr))
				return
			}
			if resolved != nil {
				props = resolved
			}
		}
		var renderOptions []inertia.RenderOption
		if config.RenderOptions != nil {
			resolved, optionsErr := config.RenderOptions(c, err, status)
			if optionsErr != nil {
				fallback(c, errors.Join(err, optionsErr))
				return
			}
			renderOptions = resolved
		}
		if renderErr := a.RenderError(c, component, props, status, renderOptions...); renderErr != nil {
			if responseCommitted(c) {
				return
			}
			fallback(c, errors.Join(err, renderErr))
		}
	}
}

func responseCommitted(c *echo.Context) bool {
	response, _ := echo.UnwrapResponse(c.Response())
	return response != nil && response.Committed
}

// DefaultErrorRenderPredicate accepts Inertia requests and non-HEAD browser
// requests that explicitly accept HTML.
func DefaultErrorRenderPredicate(c *echo.Context, err error, status int) bool {
	if c.Request().Method == http.MethodHead {
		return false
	}
	if inertia.IsInertiaRequest(c.Request()) {
		return true
	}
	accept := strings.Join(c.Request().Header.Values(echo.HeaderAccept), ",")
	for _, accepted := range strings.Split(accept, ",") {
		mediaType, params, parseErr := mime.ParseMediaType(strings.TrimSpace(accepted))
		if parseErr != nil {
			continue
		}
		if qualityValue, ok := params["q"]; ok {
			quality, qualityErr := strconv.ParseFloat(qualityValue, 64)
			if qualityErr != nil || !(quality > 0 && quality <= 1) {
				continue
			}
		}
		if mediaType == echo.MIMETextHTML || mediaType == "application/xhtml+xml" {
			return true
		}
	}
	return false
}

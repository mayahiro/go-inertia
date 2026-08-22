package inertiaecho

import (
	"net/http"

	echo "github.com/labstack/echo/v5"
)

type devToolsRouteRecorder interface {
	RecordDevToolsRoute(req *http.Request, name string, uri string, action string)
}

// Middleware returns Echo middleware backed by the underlying Inertia renderer.
func (a *Adapter) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var handlerErr error
		handler := a.Renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			c.SetRequest(req)
			c.SetResponse(w)
			defer a.recordDevToolsRoute(c)
			handlerErr = next(c)
		}))
		handler.ServeHTTP(c.Response(), c.Request())
		return handlerErr
	}
}

func (a *Adapter) recordDevToolsRoute(c *echo.Context) {
	recorder, ok := any(a.Renderer).(devToolsRouteRecorder)
	if !ok {
		return
	}
	route := c.RouteInfo()
	recorder.RecordDevToolsRoute(c.Request(), route.Name, route.Path, "")
}

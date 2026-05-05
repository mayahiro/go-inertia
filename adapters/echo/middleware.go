package inertiaecho

import (
	"net/http"

	echo "github.com/labstack/echo/v5"
)

// Middleware returns Echo middleware backed by the underlying Inertia renderer.
func (a *Adapter) Middleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var handlerErr error
		handler := a.Renderer.Middleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			c.SetRequest(req)
			c.SetResponse(w)
			handlerErr = next(c)
		}))
		handler.ServeHTTP(c.Response(), c.Request())
		return handlerErr
	}
}

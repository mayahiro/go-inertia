package inertiaecho

import (
	echo "github.com/labstack/echo/v5"
	inertia "github.com/mayahiro/go-inertia"
)

// Adapter connects an inertia.Renderer to Echo v5 handlers.
type Adapter struct {
	// Renderer is the underlying Inertia renderer.
	Renderer *inertia.Renderer
}

// New creates an Adapter for renderer.
func New(renderer *inertia.Renderer) *Adapter {
	return &Adapter{Renderer: renderer}
}

// Render renders an Inertia page through Echo.
func (a *Adapter) Render(c *echo.Context, component string, props inertia.Props, opts ...inertia.RenderOption) error {
	return a.Renderer.Render(c.Response(), c.Request(), component, props, opts...)
}

// Redirect sends an Inertia-aware redirect through Echo.
func (a *Adapter) Redirect(c *echo.Context, url string, opts ...inertia.RedirectOption) error {
	return a.Renderer.Redirect(c.Response(), c.Request(), url, opts...)
}

// Back redirects to the Echo request Referer or to "/" when no Referer is present.
func (a *Adapter) Back(c *echo.Context, opts ...inertia.RedirectOption) error {
	return a.Renderer.Back(c.Response(), c.Request(), opts...)
}

// Location sends an Inertia location response through Echo.
func (a *Adapter) Location(c *echo.Context, url string) error {
	return a.Renderer.Location(c.Response(), c.Request(), url)
}

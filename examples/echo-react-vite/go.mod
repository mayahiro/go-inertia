module github.com/mayahiro/go-inertia/examples/echo-react-vite

go 1.25.0

require (
	github.com/labstack/echo/v5 v5.1.1
	github.com/mayahiro/go-inertia v0.1.1
	github.com/mayahiro/go-inertia/adapters/echo v0.1.1
)

replace github.com/mayahiro/go-inertia => ../..

replace github.com/mayahiro/go-inertia/adapters/echo => ../../adapters/echo

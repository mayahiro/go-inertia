// Package inertia implements the server-side parts of the Inertia protocol.
//
// The package is built on net/http and has no runtime dependencies outside the
// Go standard library. Use New to create a Renderer, register Renderer.Middleware
// in your HTTP stack, and call Renderer.Render from handlers to return Inertia
// pages.
//
// Framework adapters wrap Renderer instead of reimplementing protocol behavior.
// Flash data and validation errors are exposed through FlashStore; the package
// does not include a production session store.
package inertia

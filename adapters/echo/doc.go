// Package inertiaecho adapts github.com/mayahiro/go-inertia to Echo v5.
//
// Create an Adapter with New, register Adapter.Middleware with Echo, and call
// Adapter.Render, Adapter.Redirect, Adapter.Back, or Adapter.Location from Echo
// handlers.
//
// The adapter is a thin wrapper around inertia.Renderer. Protocol behavior stays
// in the core package.
package inertiaecho

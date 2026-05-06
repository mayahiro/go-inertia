# Partial Reloads and Lazy Props

Partial reloads let the Inertia client request a subset of props for the same
page component. `go-inertia` reads the standard Inertia headers and filters
top-level props.

## Request Headers

- `X-Inertia-Partial-Component` must match the rendered component.
- `X-Inertia-Partial-Data` lists props to include.
- `X-Inertia-Partial-Except` lists props to exclude and takes precedence over
  `Partial-Data`.
- `X-Inertia-Reset` lists merge or scroll props that should replace existing
  client data instead of merging.

`errors` is always present. `flash` is included when flash data exists.

## Lazy Props

A plain `func(*http.Request) (any, error)` prop is evaluated lazily.

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"users": users,
	"companies": func(req *http.Request) (any, error) {
		return loadCompanies(req.Context())
	},
})
```

On standard visits, the callback runs. On matching partial reloads, it runs only
when the prop is included by `only` or not excluded by `except`.

You may also use `Lazy` when an explicit wrapper reads better.

```go
"companies": inertia.Lazy(loadCompanies)
```

## Optional Props

Use `Optional` for props that should never be included unless the client
explicitly asks for them with `only`.

```go
"companies": inertia.Optional(loadCompanies)
```

This is useful for secondary datasets that should not be loaded on the first
visit.

## Always Props

Use `Always` for props that should be sent even during partial reloads.

```go
"auth": inertia.Always(func(req *http.Request) (any, error) {
	return currentUser(req.Context())
})
```

`Always` is useful for page-wide state that must stay fresh, such as current
user data or feature flags.

## Composition

Lazy, optional, always, deferred, merge, and once modifiers share the same prop
model.

```go
"companies": inertia.Optional(loadCompanies).Once()
"results": inertia.Defer(loadResults).DeepMerge().MatchOn("data.id")
```

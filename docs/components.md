# Components

Inertia page components are identified by string names such as `Users/Index`.
By default, `go-inertia` sends the name passed to `Render`.

## Transforming Names

Use `Config.ComponentNameTransformer` when the server should transform component
names before they are sent to the client.

```go
renderer, err := inertia.New(inertia.Config{
	RootView: rootView,
	ComponentNameTransformer: func(component string) string {
		return component + "/Page"
	},
})
```

Calling `Render(w, req, "Users/Index", props)` now sends `Users/Index/Page` in
the page object.

## Checking Existence

Use `Config.ComponentExistenceChecker` to fail early when a component name does
not exist in the frontend application.

```go
renderer, err := inertia.New(inertia.Config{
	RootView: rootView,
	ComponentExistenceChecker: inertia.ComponentExistsFunc(
		func(component string) (bool, error) {
			_, ok := knownComponents[component]
			return ok, nil
		},
	),
})
```

When the checker returns `false`, `Render` returns an error that matches
`ErrComponentNotFound`.

```go
if errors.Is(err, inertia.ErrComponentNotFound) {
	http.NotFound(w, req)
	return
}
```

If both a transformer and checker are configured, the checker receives the
transformed component name.

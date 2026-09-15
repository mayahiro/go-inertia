# Partial Reloads and Computed Props

[日本語](partial-reloads_ja.md)

Partial reloads let the Inertia client request a subset of props for the same
page component. `go-inertia` reads the standard Inertia headers and filters
nested prop paths recursively.

## Request Headers

- `X-Inertia-Partial-Component` must match the rendered component.
- `X-Inertia-Partial-Data` lists props to include.
- `X-Inertia-Partial-Except` lists props and descendants to exclude.
- An `except`-only partial reload includes all eligible props that are not
  excluded, including optional and deferred props.
- When both filters are present, included paths must satisfy `Partial-Data` and
  must not match `Partial-Except`.
- `X-Inertia-Reset` lists merge or scroll props that should replace existing
  client data instead of merging.

`props.errors` is always present. Top-level `page.flash` is included when flash
data exists and is not filtered as a page prop.

## Nested Paths

Dot notation can request one nested branch without resolving its siblings.

```ts
router.reload({ only: ['auth.notifications'] })
```

```go
inertia.Props{
	"auth": inertia.Props{
		"user": currentUser,
		"notifications": inertia.Defer(loadNotifications),
	},
}
```

Top-level dot-notation input keys are also unpacked into nested maps and
sequential arrays before prop resolution.

```go
inertia.Props{
	"auth.user": currentUser,
	"users.0.name": "Ada",
}
```

Indexed partial paths keep their original indexes. If filtering leaves a gap,
the sparse sequence is encoded as a JSON object with numeric keys.

Use `inertia.Props`, string-key maps, slices, or arrays when nested values need
prop modifiers. Structs and custom `json.Marshaler` values remain terminal JSON
values: they can contain ordinary JSON data, but `go-inertia` does not resolve
prop modifiers inside them or filter their fields individually.

## Computed Props

A plain `func(*http.Request) (any, error)` prop is evaluated when its path is
included in the response.

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"users": users,
	"companies": func(req *http.Request) (any, error) {
		return loadCompanies(req.Context())
	},
})
```

On standard visits, the callback runs. On matching partial reloads, it runs only
when the path satisfies the active `only` and `except` filters.

Use `Computed` when an explicit wrapper reads better.

```go
"companies": inertia.Computed(loadCompanies)
```

`Lazy` is a deprecated alias for `Computed`. It is not the removed Inertia v3
`LazyProp`. Use `Optional` for props returned when selected by a partial reload.

## Optional Props

Use `Optional` for props that should be omitted from full visits and resolved
when selected by a partial reload for the same component.

```go
"companies": inertia.Optional(loadCompanies)
```

For an optional `companies` prop alongside a regular `users` prop:

| Visit | Is `companies` resolved? |
| --- | --- |
| Full visit | No |
| `only: ['companies']` | Yes |
| `only: ['users']` | No |
| `except: ['users']` | Yes |
| `except: ['companies']` | No |
| `only: ['companies'], except: ['companies']` | No |

The partial reload rows assume the component matches and neither prop has
additional modifiers. The `except` option can therefore load optional data
that has not been requested before. Use `only` to select a specific set of
optional props.

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

Computed, optional, always, deferred, merge, and once modifiers share the same prop
model.

```go
"companies": inertia.Optional(loadCompanies).Once()
"results": inertia.Defer(loadResults).DeepMerge().MatchOn("data.id")
```

## Modifier Validation

`Render` validates modifier combinations before partial filtering and returns an
error matching `ErrInvalidPropConfiguration` for combinations without protocol
meaning. `PropConfigurationError.Path` contains the full nested prop path.

Examples rejected by the renderer include `Rescue` without `Defer`, `MatchOn`
without merge or scroll behavior, `Wrapper` without scroll behavior, and extra
arguments passed to single-value variadic modifiers.

Validation also covers unrequested props so a configuration error does not stay
hidden until a later partial reload.

## Reference

See the [Inertia protocol](https://inertiajs.com/docs/v3/core-concepts/the-protocol)
for the server-side prop evaluation rules.

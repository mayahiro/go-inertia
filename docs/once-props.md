# Once Props

[日本語](once-props_ja.md)

Once props are props that the Inertia client can reuse after receiving them
once. They are useful for data that is expensive to load and changes
infrequently, such as billing plans, feature flags, or role lists.

## Server Usage

Wrap the prop with `Once`.

```go
err := renderer.Render(w, req, "Dashboard", inertia.Props{
	"plans": inertia.Once(func(req *http.Request) (any, error) {
		return loadPlans(req.Context())
	}),
})
```

The callback runs until the client reports that it already has the once key.
After that, `go-inertia` omits the prop and keeps `onceProps` metadata in the
page object.

## Custom Keys

By default, the once key is the prop name. Use `As` when a prop should share a
stable key across pages or prop names.

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"availableRoles": inertia.Once(loadRoles).As("roles"),
})
```

The page object includes:

```json
{
  "onceProps": {
    "roles": {
      "prop": "availableRoles",
      "expiresAt": null
    }
  }
}
```

## Fresh Values

Use `Fresh` to force the prop to resolve even when the client reports that it
already has the once key.

```go
"plans": inertia.Once(loadPlans).Fresh()
```

Pass `false` when the value is only conditionally fresh.

```go
"plans": inertia.Once(loadPlans).Fresh(forceRefresh)
```

## Expiration

Use `Until` to send an expiration timestamp to the client.

```go
"plans": inertia.Once(loadPlans).Until(time.Now().Add(30 * 24 * time.Hour))
```

The timestamp is serialized in `onceProps` as `expiresAt`, in Unix milliseconds.
Without `Until`, `expiresAt` is `null`.

## Composing Modifiers

Once props can be combined with deferred, merge, optional, and computed props.

```go
"permissions": inertia.Defer(loadPermissions).Once()
"activity": inertia.Merge(loadActivity).Once()
"companies": inertia.Optional(loadCompanies).Once()
```

Deferred and optional props publish `onceProps` metadata on the initial response
while omitting their values. A matching partial reload returns the values when
its `only` and `except` filters include the prop, as described below.

## Partial Reload Behavior

A matching partial reload resolves a once prop when it satisfies
`X-Inertia-Partial-Data`, if present, and is not excluded by
`X-Inertia-Partial-Except`. An `except`-only reload therefore resolves once props
that are not excluded. When both headers name the same prop, it is excluded.
These rules apply even when the client sends its key in
`X-Inertia-Except-Once-Props`, so partial reloads can refresh remembered data.

If a matching partial reload does not request the once prop, the callback is not
executed and the once metadata is omitted from that partial response.

See [Partial reloads](partial-reloads.md) for filter examples, including
optional props.

# Merge Props

Merge props tell the Inertia client to merge a prop during partial reloads
instead of replacing it. They are useful for pagination, "load more" flows, and
data that arrives in chunks.

## Root Append

Wrap a prop with `Merge` to append at the root prop path.

```go
err := renderer.Render(w, req, "Items/Index", inertia.Props{
	"items": inertia.Merge(items),
})
```

The page object includes:

```json
{
  "mergeProps": ["items"]
}
```

## Nested Paths

Use `Append` and `Prepend` to target relative paths inside the prop. `go-inertia`
prefixes these paths with the page prop name.

```go
err := renderer.Render(w, req, "Forum/Index", inertia.Props{
	"forum": inertia.Merge(forumData).
		Append("posts").
		Prepend("announcements"),
})
```

The page object includes:

```json
{
  "mergeProps": ["forum.posts"],
  "prependProps": ["forum.announcements"]
}
```

## Matching Items

Use `MatchOn` to tell the client how to match existing items while merging
arrays. Paths are relative to the prop and are serialized as full page prop
paths.

```go
"results": inertia.Merge(results).Append("data").MatchOn("data.id")
```

The page object includes:

```json
{
  "mergeProps": ["results.data"],
  "matchPropsOn": ["results.data.id"]
}
```

## Deep Merge

Use `DeepMerge` when the whole prop should be deeply merged.

```go
"chat": inertia.Merge(chat).DeepMerge().MatchOn("messages.id")
```

The page object includes:

```json
{
  "deepMergeProps": ["chat"],
  "matchPropsOn": ["chat.messages.id"]
}
```

## Computed Values

`Merge` accepts either a value or a `func(*http.Request) (any, error)`.

```go
"items": inertia.Merge(func(req *http.Request) (any, error) {
	return loadItems(req.Context())
})
```

This callback is resolved when the page is rendered. It is not the same as a
deferred prop. During matching partial reloads, the callback only runs when the
prop is included by `only` or not excluded by `except`.

## Composing Modifiers

Merge props can be combined with `Once` when a mergeable prop should be resolved
once and then reused by the client on later visits.

```go
"activity": inertia.Merge(loadActivity).Once()
```

Use `Defer` before merge modifiers when the first page response should omit the
prop and load it after mount.

```go
"results": inertia.Defer(loadResults).Append("data").MatchOn("data.id")
"chat": inertia.Defer(loadChat).DeepMerge().MatchOn("messages.id")
```

## Resetting Props

When the client sends `X-Inertia-Reset`, `go-inertia` keeps the prop value in
the response but removes merge metadata for that prop. The client then replaces
the prop instead of merging it.

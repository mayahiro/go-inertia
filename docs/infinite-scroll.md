# Infinite Scroll

Infinite scroll props are paginated props rendered with the Inertia client
`InfiniteScroll` component. The server response must include the prop value,
merge metadata for the wrapped item collection, and `scrollProps` pagination
metadata.

## Server Usage

Wrap the paginated prop with `Scroll`.

```go
err := renderer.Render(w, req, "Posts/Index", inertia.Props{
	"posts": inertia.Scroll(inertia.Props{
		"data": posts,
	}, inertia.ScrollMetadata{
		PreviousPage: nil,
		NextPage:     2,
		CurrentPage:  1,
	}),
})
```

`PageName` defaults to `page` when it is empty.

```go
"posts": inertia.Scroll(inertia.Props{
	"data": posts,
}, inertia.ScrollMetadata{
	PageName:     "posts",
	PreviousPage: previousPage,
	NextPage:     nextPage,
	CurrentPage:  currentPage,
})
```

The resulting page object includes `mergeProps` and `scrollProps`.

```json
{
  "mergeProps": ["posts.data"],
  "scrollProps": {
    "posts": {
      "pageName": "page",
      "previousPage": null,
      "nextPage": 2,
      "currentPage": 1
    }
  }
}
```

## Data Wrapper

`Scroll` merges the `data` wrapper by default because Inertia paginated props
commonly use a `data` array. Use `Wrapper` when your prop uses another item
wrapper.

```go
"feed": inertia.Scroll(inertia.Props{
	"items": items,
}, metadata).Wrapper("items")
```

This emits:

```json
{
  "mergeProps": ["feed.items"]
}
```

## Matching Items

Use `MatchOn` when the client should match existing items by an identifier
while merging. Paths are relative to the page prop and are serialized as full
page prop paths.

```go
"feed": inertia.Scroll(feed, metadata).Wrapper("items").MatchOn("items.id")
```

The page object includes:

```json
{
  "matchPropsOn": ["feed.items.id"]
}
```

## Loading Direction

The Inertia client sends `X-Inertia-Infinite-Scroll-Merge-Intent` when loading
additional scroll data. `go-inertia` appends by default. When the header value
is `prepend`, it emits `prependProps` instead.

```json
{
  "prependProps": ["posts.data"]
}
```

Application handlers do not need to read this header directly unless they need
custom query behavior. The `Scroll` prop reads it while building page metadata.

## Resetting

When filters or search parameters change, use the Inertia client `reset` visit
option for the scroll prop. The server keeps the prop value in the response,
removes merge metadata, and marks the scroll metadata as reset.

```json
{
  "scrollProps": {
    "posts": {
      "pageName": "page",
      "previousPage": null,
      "nextPage": 2,
      "currentPage": 1,
      "reset": true
    }
  }
}
```

`go-inertia` does not reset an infinite scroll prop by itself after `POST`,
`PUT`, `PATCH`, or `DELETE` requests. It only reacts to the protocol headers
sent by the Inertia client. A scroll prop is reset when the client visit sends
the prop name in the `reset` option.

```tsx
form.post("/users", {
  reset: ["users"],
  preserveState: "errors",
})
```

Use this when a successful mutation should replace the loaded scroll data with
fresh server data, such as returning to the first page after filters change or
after creating an item that should appear at the top of the list.

If a form lives on the same page as a long infinite scroll list and the user
should stay on the currently loaded list, do not send `reset` for that prop.
Inertia `post`, `put`, `patch`, and `delete` visits preserve component state by
default. In that case, applications often keep the current scroll state and
update the list with client-side prop helpers such as `router.prependToProp` or
`router.appendToProp`, or reload only the data they need later.

## Client Usage

Use the client adapter's `InfiniteScroll` component with the prop name.

```tsx
import { InfiniteScroll } from "@inertiajs/react"

export default function PostsIndex({ posts }) {
  return (
    <InfiniteScroll data="posts">
      {posts.data.map((post) => (
        <article key={post.id}>{post.title}</article>
      ))}
    </InfiniteScroll>
  )
}
```

## Current Limitations

`go-inertia` expects applications to pass `ScrollMetadata` explicitly. It does
not inspect or normalize paginator structs from a database library.

`Scroll` uses the same internal prop modifier model as `Defer`, `Merge`, and
`Once`, but paginated scroll props already carry merge metadata. Prefer plain
`Scroll` unless a specific page has a tested need for additional modifiers.

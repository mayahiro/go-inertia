# Server-Provided Head Elements

Inertia v3.5 and later can synchronize server-provided head elements across
client navigations. `go-inertia` provides a structured builder that escapes
element text and attribute values before sending the raw HTML strings expected
by the client.

## Server Setup

Build the head collection and pass it to `Render` with `WithServerHead`.

```go
head, err := inertia.NewServerHead(
	inertia.HeadTitle("Users | Admin"),
	inertia.HeadMeta("description", "Manage users"),
	inertia.HeadMetaProperty("og:title", "Users | Admin"),
	inertia.HeadLink("canonical", "https://example.com/users"),
	inertia.HeadScript("/build/analytics.js").Attr("defer", "defer"),
)
if err != nil {
	return err
}

return renderer.Render(w, req, "Users/Index", props,
	inertia.WithServerHead(head),
)
```

The default wire prop is `page.props.head`. It is marked as an always prop so
partial reloads can also update the managed document head. The same elements
are appended to `RootViewData.InertiaHead` for the initial browser response.

## Client Setup

Enable the Inertia client option in `createInertiaApp`.

```ts
createInertiaApp({
  serverHead: true,
  // ...
})
```

Use `@inertiajs/core` and the framework adapter from the same Inertia release.

## Custom Prop Name

Set the server and client to the same custom top-level prop name.

```go
renderer, err := inertia.New(inertia.Config{
	RootView:       rootView,
	ServerHeadProp: "seoHead",
})
```

```ts
createInertiaApp({
  serverHead: 'seoHead',
  // ...
})
```

`ServerHead.NamedProps` is available when an application needs the generated
string array without `WithServerHead`. That lower-level path does not
automatically add the initial HTML fallback.

## Stable Keys

The structured helpers add stable `data-inertia` values. Override a generated
key with `Key` when several elements have application-specific identities.

```go
inertia.HeadMeta("description", description).Key("page-description")
```

`Key` requires a non-empty value so structured elements cannot accidentally
share an empty identity.

Trusted raw elements without `data-inertia` receive a positional
`server-head-N` key so the initial fallback and client-managed element have the
same identity.

Text and attribute values are HTML-escaped. Event-handler attributes such as
`onload` are rejected. URL values still describe resources the browser may
load, so construct script and link URLs from trusted application data.

## Trusted Raw HTML

Use `RawHeadElement` only for HTML that is already trusted and reviewed.

```go
head, err := inertia.NewServerHead(
	inertia.RawHeadElement(template.HTML(`<meta name="custom" content="trusted">`)),
)
```

Pass one complete head element per `RawHeadElement` call. The `template.HTML`
argument and the method name make the trust boundary explicit. Never convert
user input to `template.HTML`.

## `WithInertiaHead` Is Different

`WithInertiaHead` only supplies root-template HTML for the current initial
response. It does not create a page prop and cannot synchronize changes during
Inertia navigation. Use `WithServerHead` for Inertia v3 server-provided head
elements.

This feature does not implement Inertia SSR. It provides head elements for the
normal client-side rendering flow and an initial root-template fallback.

# History Flags

The Inertia page object supports `encryptHistory` and `clearHistory`.
`go-inertia` exposes these as render options.

## Encrypt History

Use `WithEncryptHistory` when the current page should request encrypted browser
history state from clients that support it.

```go
return renderer.Render(w, req, "Account/Security", props,
	inertia.WithEncryptHistory(),
)
```

The rendered page object includes:

```json
{
  "encryptHistory": true
}
```

## Clear History

Use `WithClearHistory` when the current response should request clearing stored
history state from clients that support it.

```go
return renderer.Render(w, req, "Auth/Login", props,
	inertia.WithClearHistory(),
)
```

The rendered page object includes:

```json
{
  "clearHistory": true
}
```

Both options can be passed through `Config.DefaultRenderOptions` when an
application wants to apply one of them broadly.

Pass `false` to disable a default option for one render call.

```go
return renderer.Render(w, req, "Public/Page", props,
	inertia.WithEncryptHistory(false),
)
```

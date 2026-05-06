# File Uploads

Inertia client adapters automatically send `FormData` when a form contains
files. `go-inertia` does not parse uploads itself; use the request helpers from
`net/http` or your framework adapter.

## net/http

```go
func storeAvatar(w http.ResponseWriter, req *http.Request) {
	file, header, err := req.FormFile("avatar")
	if err != nil {
		_ = renderer.Back(w, req, inertia.WithValidationErrors(inertia.ValidationErrors{
			"avatar": "Avatar is required",
		}))
		return
	}
	defer file.Close()

	_ = header
	_ = renderer.Redirect(w, req, "/profile", inertia.WithFlash(inertia.Flash{
		"success": "Avatar uploaded",
	}))
}
```

## Echo

```go
e.POST("/profile/avatar", func(c *echo.Context) error {
	file, err := c.FormFile("avatar")
	if err != nil {
		return app.Back(c, inertia.WithValidationErrors(inertia.ValidationErrors{
			"avatar": "Avatar is required",
		}))
	}

	_ = file
	return app.Redirect(c, "/profile", inertia.WithFlash(inertia.Flash{
		"success": "Avatar uploaded",
	}))
})
```

## Validation Flow

For normal Inertia submissions, return a redirect with flashed validation
errors. Do not return `422` JSON for standard Inertia form submissions.

Precognition is the exception: validation-only requests should use
`PrecognitionSuccess` or `PrecognitionErrors`.

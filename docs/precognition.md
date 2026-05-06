# Precognition

Precognition is a validation-only request flow. The client sends a request that
matches the eventual form submission, but the server validates the input without
running the mutation.

Normal Inertia validation still uses redirects and flashed errors. Precognition
uses direct validation responses instead.

## Request Helpers

Use `IsPrecognition` to detect a Precognition validation request.

```go
if inertia.IsPrecognition(req) {
	// Run validation only.
}
```

When the client validates selected fields, read them with
`PrecognitionValidateOnly`.

```go
fields := inertia.PrecognitionValidateOnly(req)
```

The helper reads `Precognition-Validate-Only` as a comma-separated list.

## Successful Validation

Write a successful validation response with `PrecognitionSuccess`.

```go
inertia.PrecognitionSuccess(w)
return nil
```

The response is `204 No Content` and includes the required Precognition
headers.

## Failed Validation

Write validation errors with `PrecognitionErrors`.

```go
return inertia.PrecognitionErrors(w, inertia.ValidationErrors{
	"email": []string{"Email is required"},
})
```

The response is `422 Unprocessable Entity` with this shape:

```json
{
  "errors": {
    "email": ["Email is required"]
  }
}
```

## Side Effects

Precognition handlers should not create records, enqueue jobs, send email, or
perform other mutation side effects. Middleware that tracks analytics or
interactions can use `IsPrecognition(req)` to skip that work.

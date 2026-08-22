# Echo Adapter

[English](echo.md)

Echo v5 adapterは別moduleとして提供する

```txt
github.com/mayahiro/go-inertia/adapters/echo
```

core packageはEchoをimportしない

## 要件

- Go 1.25.0以降
- Echo v5.3.1以降
- go-inertia v0.5.0

## インストール

```sh
go get github.com/mayahiro/go-inertia/adapters/echo
```

## セットアップ

core rendererを作成してEcho adapterでwrapし、adapter middlewareを登録する
`Renderer.Middleware`は`net/http` middlewareだが、Echo adapterは`e.Use`で直接使える`app.Middleware`を公開する
applicationから`echo.WrapMiddleware`を呼ぶ必要はない

```go
package main

import (
	"net/http"

	echo "github.com/labstack/echo/v5"
	inertia "github.com/mayahiro/go-inertia"
	inertiaecho "github.com/mayahiro/go-inertia/adapters/echo"
)

func main() {
	rootView, err := inertia.NewTemplateRootViewFromFile("views/app.html", "app.html")
	if err != nil {
		panic(err)
	}

	renderer, err := inertia.New(inertia.Config{
		RootView:   rootView,
		FlashStore: inertia.NewMemoryFlashStore(),
	})
	if err != nil {
		panic(err)
	}

	app := inertiaecho.New(renderer)
	e := echo.New()
	e.HTTPErrorHandler = app.ErrorHandler(inertiaecho.ErrorHandlerConfig{
		Pages: map[int]string{
			http.StatusNotFound: "Errors/NotFound",
		},
		DefaultComponent: "Errors/Error",
		Props: func(c *echo.Context, err error, status int) (inertia.Props, error) {
			return inertia.Props{"status": status}, nil
		},
	})
	e.Use(app.Middleware)

	e.GET("/", func(c *echo.Context) error {
		return app.Render(c, "Dashboard", inertia.Props{
			"message": "Hello",
		})
	})

	e.POST("/users", func(c *echo.Context) error {
		return app.Redirect(c, "/users", inertia.WithFlash(inertia.Flash{
			"success": "User created",
		}))
	})

	if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
		e.Logger.Error("server error", "error", err)
	}
}
```

## Handler helper

adapterはEcho向けに次のmethodを公開する

- `Render`
- `RenderError`
- `Redirect`
- `Back`
- `Location`

これらのmethodはprotocol処理を内部のcore `Renderer`へ委譲する
status別のerror pageには`RenderError`を使い、pageで既定以外のresponse処理が必要な場合は`inertia.WithRenderStatus`などのrender optionを`Render`へ渡す

## Error handler

`Adapter.ErrorHandler`はEcho errorをInertia componentへ対応付け、Echoの集中error handlerとして設定できる

```go
e.HTTPErrorHandler = app.ErrorHandler(inertiaecho.ErrorHandlerConfig{
	Pages: map[int]string{
		http.StatusNotFound: "Errors/NotFound",
		http.StatusForbidden: "Errors/Forbidden",
	},
	DefaultComponent: "Errors/Error",
	Props: func(c *echo.Context, err error, status int) (inertia.Props, error) {
		return inertia.Props{"status": status}, nil
	},
})
```

既定のpredicateはInertia requestと、`Accept` headerに`text/html`または`application/xhtml+xml`を明示したbrowser requestをrenderする
`HEAD`、wildcardだけのrequest、HTML以外のrequestにはfallback handlerを使い、既定ではEchoのJSON error handlerへ委譲する
これにより、APIとstatic requestはHTMLを明示的に受け付けない限りInertia pageを受け取らない
error handlerを使うresponseは`Accept`と`X-Inertia`で変化する

route groupごとに異なるpolicyが必要な場合は`ShouldRender`を上書きし、またはcustom `Fallback`を設定する
`Props`では元のerrorから安全に公開できる値を生成できる
core rendererで設定したshared propsも引き続き適用する
custom policyから`DefaultErrorRenderPredicate`を呼ぶと、既定のcontent negotiationを維持しながらapplication固有の条件を追加できる
Echoの既定fallbackはerrorをlogへ記録しないため、集中loggingが必要な場合はlogging middlewareまたはcustom fallbackを使う

server-provided head elementなどerrorごとのrender設定には`RenderOptions`を使う

```go
RenderOptions: func(c *echo.Context, err error, status int) ([]inertia.RenderOption, error) {
	head, buildErr := inertia.NewServerHead(
		inertia.HeadTitle(http.StatusText(status)),
	)
	return []inertia.RenderOption{inertia.WithServerHead(head)}, buildErr
},
```

prop生成、render option生成、renderのいずれかが失敗した場合、設定したfallbackは元のerrorと生成中のerrorを受け取る
Echo handlerがすでにcommitしたresponseは変更しない

## Echo v4

公開しているEcho adapterはEcho v5を対象とする
将来Echo v4へ対応する場合は別のadapter moduleとして提供する

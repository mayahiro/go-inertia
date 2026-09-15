# go-inertia

[English](README.md)

Go 向けの小さな Inertia.js サーバーアダプター

core package は `net/http` を基盤とし、Go 標準ライブラリ以外の実行時依存を持たない
Echo v5 対応は別の adapter module として提供する

## 状態

このプロジェクトは v0 系を提供している
利用できるバージョンは Git tag を確認すること

ドキュメントと example は Inertia.js 3.x のクライアントパッケージを対象とする
`go-inertia` は後述のプロトコル機能を実装するが、公式 Laravel adapter の全機能を代替するものではない

renderer はクライアント側で描画する Inertia アプリケーションを対象とする
Inertia SSR の gateway、body、head の描画は未実装

## パッケージ構成

- `github.com/mayahiro/go-inertia`: フレームワークに依存しない core package
- `github.com/mayahiro/go-inertia/adapters/echo`: core renderer を Echo v5 に接続する adapter
- `examples/echo-react-vite`: 独立した module の example アプリケーション

## 要件

- core package: Go 1.25.0 以降
- Echo adapter: Echo v5 の要件により Go 1.25.0 以降
- example の frontend: Node.js 24 以降

## インストール

```sh
go get github.com/mayahiro/go-inertia
```

Echo v5 を使う場合:

```sh
go get github.com/mayahiro/go-inertia/adapters/echo
```

## 提供する機能

- 初回訪問の HTML レスポンス
- Inertia JSON レスポンス
- `Vary: X-Inertia`
- アセットのバージョン不一致への対応
- Inertia リダイレクト、戻るリダイレクト、外部への location レスポンス
- サーバー側の共有 props と `sharedProps` metadata
- エラーページの HTTP status 指定
- コンポーネント名の変換と存在確認
- flash データと検証エラーのインターフェース
- 単一プロセス向けのメモリー内 flash store
- 永続セッションと flash の連携
- ネストした props の再帰的な解決とドット記法の部分リロード
- computed、optional、always props
- deferred props と `rescuedProps` metadata
- once props
- merge、prepend、deep merge props
- deferred、once、merge の modifier の組み合わせ
- deferred と once を組み合わせた無限スクロール props
- 無限スクロール props と paginator helper
- Precognition の request と response helper
- 履歴の暗号化と履歴消去フラグ
- prefetch リクエストの検出
- backend に依存しない Inertia DevTools の記録と読み取り endpoint
- Vite manifest、import した chunk、dev server 用タグの生成
- 既定の render option
- サーバーから提供する構造化された head 要素
- クライアントの root element id の設定と root template helper
- Echo v5 adapter
- 設定可能な Echo v5 の集中エラーハンドラー
- endpoint のテスト helper

## 組み込み時の注意

- Inertia ページを描画するルートより前に `Renderer.Middleware` またはフレームワークの adapter middleware を登録する
- `Props`、共有 props、flash データ、検証エラーはブラウザーへ送られるため、秘密情報を入れない
- 大きなページではページ専用の Go struct を定義し、描画の境界で `inertia.Props` に変換すると、server と frontend の受け渡しを確認しやすい
- `NewMemoryFlashStore` はローカル開発、テスト、単一プロセスの example 向け。本番環境や複数プロセスのアプリケーションでは `NewSessionFlashStore` で既存の永続セッションに接続するか、共有 backend 用に `FlashStore` を実装する
- `go build` は Go コードだけをビルドする。アプリケーションに埋め込まない場合、テンプレートと Vite assets はファイルとして配布する

## DevTools

`Config.DevTools` でローカル recorder を有効にし、`createInertiaApp({ dev: import.meta.env.DEV })` で対応するクライアント hook を有効にする
recorder は既定で無効で、メモリー内の store を使用し、一般的な秘密情報のキーとヘッダーを伏せる
明示的な `Authorize` callback を設定しない場合は、直接の loopback リクエストだけを受け付ける

```go
renderer, err := inertia.New(inertia.Config{
	RootView: rootView,
	DevTools: inertia.DevToolsConfig{
		Enabled: os.Getenv("INERTIA_DEVTOOLS_ENABLED") == "true",
		ComponentPathResolver: func(component string) string {
			return "resources/js/Pages/" + component + ".tsx"
		},
	},
})
```

設定、セキュリティ上の境界、保存制限、フレームワークの metadata hook は [DevTools](docs/devtools_ja.md) を参照

## Core の使用例

```go
package main

import (
	"net/http"

	inertia "github.com/mayahiro/go-inertia"
)

func main() {
	rootView, err := inertia.NewTemplateRootViewFromFile("views/app.html", "app.html")
	if err != nil {
		panic(err)
	}

	renderer, err := inertia.New(inertia.Config{
		RootView: rootView,
		SharedProps: inertia.StaticSharedProps(inertia.Props{
			"app": map[string]any{"name": "Admin"},
		}),
	})
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		err := renderer.Render(w, req, "Dashboard", inertia.Props{
			"message": "Hello",
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.ListenAndServe(":8080", renderer.Middleware(mux))
}
```

## Root Template

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    {{ .ViteTags }}
    {{ .InertiaHead }}
  </head>
  <body>
    {{ .InertiaApp }}
  </body>
</html>
```

`InertiaApp` は初期ページの script とマウント要素を `Config.RootElementID` に合わせて出力する
`app` 以外の値を使う場合は、`createInertiaApp` に同じ `id` を設定する

## サーバーから提供する Head 要素

escape 済みの head 要素を作成し、`WithServerHead` に渡す
初回ドキュメントに描画され、後続のクライアント遷移では `page.props.head` として送信される

```go
head, err := inertia.NewServerHead(
	inertia.HeadTitle("Users"),
	inertia.HeadMeta("description", "Manage users"),
)
if err != nil {
	return err
}

return renderer.Render(w, req, "Users/Index", props,
	inertia.WithServerHead(head),
)
```

対応する Inertia v3 のクライアント設定を有効にする

```ts
createInertiaApp({
  serverHead: true,
  // ...
})
```

独自の prop 名、安定したキー、信頼できる生の HTML は[サーバーから提供する Head 要素](docs/server-head.md)を参照

## Echo の使用例

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

## Vite

`NewVite` を使うと Vite の entrypoint 用のタグを生成できる

開発モード:

```go
vite, err := inertia.NewVite(inertia.ViteConfig{
	DevServerURL: "http://127.0.0.1:5173",
	Entry:        "resources/js/app.tsx",
	ReactRefresh: true,
})
```

本番モード:

```go
vite, err := inertia.NewVite(inertia.ViteConfig{
	ManifestPath: "public/build/.vite/manifest.json",
	PublicPath:   "/build",
	Entry:        "resources/js/app.tsx",
})
```

すべてのページで同じ root template の assets を使う場合は、生成したタグを既定の render option に設定する

```go
tags, err := vite.Tags()
if err != nil {
	return err
}

renderer, err := inertia.New(inertia.Config{
	RootView:        rootView,
	VersionProvider: vite.VersionProvider(),
	DefaultRenderOptions: []inertia.RenderOption{
		inertia.WithViteTags(tags),
	},
})
```

特定のリクエストで既定のタグを上書きする場合は、`Render` に `inertia.WithViteTags(tags)` を渡すこともできる

## Flash と検証

Inertia の検証では通常、`422` の JSON レスポンスを返さず、戻るリダイレクトと flash に保存した検証エラーを使用する
`go-inertia` は `FlashStore` インターフェースと、開発用の小さなメモリー内実装を提供する

```go
renderer, err := inertia.New(inertia.Config{
	RootView:   rootView,
	FlashStore: inertia.NewMemoryFlashStore(),
})
```

検証に失敗した場合は `Back` と `WithValidationErrors` を使う

```go
return renderer.Back(w, req, inertia.WithValidationErrors(inertia.ValidationErrors{
	"name": "Name is required",
}))
```

Inertia は GET 以外のリクエスト後にコンポーネントの state を保持するため、通常は元の入力値をサーバーの props で送り直す必要はない

## Computed、Optional、Always Props

通常の `func(*http.Request) (any, error)` props はレスポンスに含まれる場合に評価される
コンポーネントが一致する部分リロードでは、適用中の `only` を満たし、`except` によって除外されない場合に callback が実行される

```go
"companies": func(req *http.Request) (any, error) {
	return loadCompanies(req.Context())
}
```

明示的な wrapper の方が分かりやすい場合は `Computed` を使う

```go
"companies": inertia.Computed(loadCompanies)
```

`Lazy` は `Computed` の非推奨 alias として残している
Inertia v3 で削除された `LazyProp` とは異なり、部分リロードで選択された場合に返す props には `Optional` を使う

`Optional` は通常の訪問では省略され、同じコンポーネントへの部分リロードで選択された場合に解決される
クライアントの `only` は指定した props を選択し、`except` だけの部分リロードは除外されていない optional props を含める
両方のフィルターがある場合、prop は `only` を満たし、`except` に一致しない必要がある

```go
"companies": inertia.Optional(loadCompanies)
```

部分リロードでも返したい props には `Always` を使う

```go
"auth": inertia.Always(currentUser)
```

## Deferred Props

初回ページの描画後に取得する props には `Defer` を使う
クライアントが部分リロードで prop を要求した場合に callback が実行される

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"users": UserList(),
	"permissions": inertia.Defer(func(req *http.Request) (any, error) {
		return PermissionList(req.Context())
	}),
})
```

deferred props は既定で `default` グループに属する
複数の props を別のリクエストグループで取得する場合はグループ名を指定する

```go
"teams": inertia.Defer(loadTeams, "attributes")
```

取得時のエラーでレスポンス全体を失敗させず、クライアントの `<Deferred>` に rescue 用の表示をさせる場合は `Rescue` を使う

```go
"permissions": inertia.Defer(loadPermissions).Rescue()
```

## Once Props

最初に受け取った値をクライアントが再利用できる props には `Once` を使う

```go
err := renderer.Render(w, req, "Dashboard", inertia.Props{
	"plans": inertia.Once(func(req *http.Request) (any, error) {
		return BillingPlans(req.Context())
	}),
})
```

prop 名をまたいで once key を共有するには `As`、強制的に再取得するには `Fresh`、有効期限を送るには `Until` を使う

```go
"availableRoles": inertia.Once(loadRoles).As("roles")
```

## Merge Props

部分リロードで値を追記する props には `Merge` を使う

```go
err := renderer.Render(w, req, "Items/Index", inertia.Props{
	"items": inertia.Merge(items),
})
```

`Append` と `Prepend` でネストしたパスを指定できる
単純な追加ではなく識別子で既存の要素と対応付ける場合は `MatchOn` を使う

```go
"results": inertia.Merge(results).Append("data").MatchOn("data.id")
```

prop 全体を深くマージする場合は `DeepMerge` を使う

```go
"chat": inertia.Merge(chat).DeepMerge().MatchOn("messages.id")
```

## Prop Modifier の組み合わせ

`Defer`、`Merge`、`Once`、`Optional`、`Always`、computed props は共通の modifier モデルを使用する
Inertia プロトコルが対応する組み合わせを使用できる

```go
"results": inertia.Defer(loadResults).DeepMerge().MatchOn("data.id")
"permissions": inertia.Defer(loadPermissions).Once()
"activity": inertia.Merge(loadActivity).Once()
"companies": inertia.Optional(loadCompanies).Once()
```

## 無限スクロール

Inertia クライアントの `InfiniteScroll` コンポーネントに渡すページ分割された props には `Scroll` を使う
値とページ情報の metadata を明示的に渡すか、アプリケーションの paginator を `ScrollPaginator` に対応させて `ScrollPage` に渡す

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

`Scroll` は既定で `data` wrapper をマージし、`scrollProps` metadata を設定する
別の wrapper を使う場合は `Wrapper`、識別子で要素を対応付ける場合は `MatchOn` を使う

```go
"feed": inertia.Scroll(feed, metadata).Wrapper("items").MatchOn("items.id")
```

`ScrollPage` は既定で paginator の要素を `data` に格納する
ページオブジェクトの要素キーが異なる場合は、独自の wrapper 名を指定できる

```go
"users": inertia.ScrollPage(userPage)
"feed": inertia.ScrollPage(feedPage, "items")
```

`go-inertia` は任意のデータベース paginator struct を reflection で読み取らない
小さな `ScrollPaginator` インターフェースか明示的な `ScrollMetadata` を使い、ページ情報の形式をアプリケーション側で管理する

`Scroll` は Inertia プロトコルに従い、フォーム送信後に取得済みのスクロールデータを自動ではリセットしない
更新成功時に最初のページから取得し直す場合は、クライアントの visit option に `reset` を指定する
現在の取得済み一覧を保持する場合は `reset` を省略する

## Precognition

Precognition は検証だけを行うリクエスト
`IsPrecognition` と `PrecognitionValidateOnly` を使って検出し、検証の処理範囲を絞る

```go
if inertia.IsPrecognition(req) {
	if len(errors) > 0 {
		return inertia.PrecognitionErrors(w, errors)
	}
	inertia.PrecognitionSuccess(w)
	return nil
}
```

Precognition helper は通常の Inertia の検証とは独立している
通常のフォーム送信では引き続きリダイレクトと flash の検証エラーを使う

## 履歴フラグ

render option で Inertia のページレスポンスに履歴フラグを設定できる

```go
return renderer.Render(w, req, "Account/Security", props,
	inertia.WithEncryptHistory(),
	inertia.WithClearHistory(),
)
```

## React + Vite の使用例

TypeScript、React、Vite、Echo の使用例は [examples/echo-react-vite](examples/echo-react-vite) を参照

## ドキュメント

- [使い始める](docs/getting-started.md)
- [プロトコル](docs/protocol_ja.md)
- [コンポーネント](docs/components.md)
- [Echo adapter](docs/echo_ja.md)
- [Vite](docs/vite.md)
- [検証と flash](docs/validation-and-flash.md)
- [サーバーから提供する Head 要素](docs/server-head.md)
- [部分リロードと Computed Props](docs/partial-reloads_ja.md)
- [Precognition](docs/precognition.md)
- [履歴フラグ](docs/history.md)
- [ファイルアップロード](docs/file-uploads.md)
- [Deferred Props](docs/deferred-props.md)
- [Once Props](docs/once-props_ja.md)
- [Merge Props](docs/merge-props.md)
- [無限スクロール](docs/infinite-scroll.md)
- [テスト](docs/testing.md)
- [DevTools](docs/devtools_ja.md)

## 公開 Helper の対象外

次の Inertia の用途は現在の公開 API の対象外:

- SSR gateway、SSR body、SSR head を含むサーバー側の描画
- Redis、データベース、フレームワーク固有のセッションとの具体的な連携
- Echo v4 adapter
- Echo v5 以外のフレームワーク用 adapter
- CLI によるひな形の生成

## 開発

Go の import と整形は `goimports` で行う
ツールへの依存は独立した `tools` module に置き、公開する root module は依存なしに保つ

```sh
cd tools
go tool goimports -w ..
```

core の確認:

```sh
go test ./...
go vet ./...
```

Echo adapter の確認:

```sh
cd adapters/echo
go test ./...
go vet ./...
```

example の確認:

```sh
cd examples/echo-react-vite
npm ci
npm run build
go test ./...
go vet ./...
```

## 参照

- Inertia プロトコル: https://inertiajs.com/docs/v3/core-concepts/the-protocol
- Inertia リダイレクト: https://inertiajs.com/docs/v3/the-basics/redirects
- Inertia 検証: https://inertiajs.com/docs/v3/the-basics/validation
- Inertia 共有データ: https://inertiajs.com/docs/v3/data-props/shared-data
- Inertia 部分リロード: https://inertiajs.com/docs/v3/data-props/partial-reloads
- Inertia deferred props: https://inertiajs.com/docs/v3/data-props/deferred-props
- Inertia once props: https://inertiajs.com/docs/v3/data-props/once-props
- Inertia アセットのバージョン管理: https://inertiajs.com/docs/v3/advanced/asset-versioning
- Inertia props のマージ: https://inertiajs.com/docs/v3/data-props/merging-props
- Inertia 無限スクロール: https://inertiajs.com/docs/v3/data-props/infinite-scroll
- Inertia instant visit: https://inertiajs.com/docs/v3/the-basics/instant-visits
- Inertia フォーム: https://inertiajs.com/docs/v3/the-basics/forms
- Inertia ファイルアップロード: https://inertiajs.com/docs/v3/the-basics/file-uploads
- Inertia 履歴暗号化: https://inertiajs.com/docs/v3/security/history-encryption
- Laravel Precognition: https://laravel.com/docs/13.x/precognition
- Vite backend 連携: https://vite.dev/guide/backend-integration.html
- Echo: https://echo.labstack.com/

## ライセンス

MIT

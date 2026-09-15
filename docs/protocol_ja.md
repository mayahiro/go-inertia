# プロトコル

[English](protocol.md)

`go-inertia` は、Inertia の基本プロトコルに必要なサーバー側の処理を実装する
対象は初回 HTML、Inertia JSON、アセットのバージョン不一致、リダイレクト、共有 props の合成、flash、検証エラー、再帰的な prop 解決、ドット記法の部分リロード、computed、optional、always、deferred、once、merge、modifier の組み合わせ、無限スクロール、サーバーから提供する head 要素、履歴フラグ、prefetch の検出、Precognition の検証レスポンス

## 初回の HTML

通常のブラウザー訪問には、設定した `RootView` が描画した HTML ドキュメントを返す
ドキュメントには安全な JSON 形式のページデータと、クライアントアプリをマウントする要素を含める

```html
<script data-page="app" type="application/json">...</script>
<div id="app"></div>
```

レスポンスには `Vary: X-Inertia` を含める

`Config.RootElementID` で両方の値を変更できる
テンプレートでは `InertiaApp` を使い、対応する script とマウント要素をまとめて出力できる

## Inertia JSON の訪問

`X-Inertia: true` を持つリクエストには JSON 形式のページオブジェクトを返す
レスポンスには次のヘッダーを設定する:

- `X-Inertia: true`
- `Content-Type: application/json`
- `Vary: X-Inertia`

## ページオブジェクト

ページオブジェクトは次の基本フィールドをサポートする:

- `component`
- `props`
- `url`
- `version`
- `encryptHistory`
- `clearHistory`
- `preserveFragment`
- `flash`: 一時的な flash データがある場合

高度な props の metadata として、次の JSON フィールドも使用する:

- `mergeProps`
- `prependProps`
- `deepMergeProps`
- `matchPropsOn`
- `scrollProps`
- `deferredProps`
- `rescuedProps`
- `sharedProps`
- `onceProps`

これらの metadata は現在の Inertia クライアントが受け付けるプロトコルの形式で出力する

`version` は常に含まれる
アセットのバージョン管理を設定しない場合、既定の provider は空文字列を返す

`props.errors` は常に含まれる
検証エラーがない場合は空オブジェクトになる

`sharedProps` は `Config.SharedProps` または `WithSharedProps` で登録したトップレベルの prop 名を列挙する
handler が最終的な値を上書きしてもキーは共有 metadata に残り、Inertia v3 の resolver の仕様と一致する

flash データはトップレベルの `page.flash` に格納する
`page.props.flash` に格納しないため、Inertia v3 クライアントは通常の prop として履歴に保存せず、flash の callback とイベントを発行できる

## アセットのバージョン不一致

空でないアセットバージョンを設定すると、middleware は GET の Inertia リクエストで `X-Inertia-Version` と比較する
不一致の場合は `409 Conflict` を返し、`X-Inertia-Location` に現在の URL、`X-Inertia-Version` に現在のバージョンを設定する
Inertia v3.6 以降のクライアントはこのレスポンスバージョンにより、アセット変更と明示的な location レスポンスを区別し、バックグラウンドリクエストに起因するページ全体の再読み込みを遅らせられる

GET 以外のリクエストには、バージョン不一致のレスポンスを直接返さない

## リダイレクト

GET 以外の Inertia リダイレクトには `303 See Other` を使用する
外部への location レスポンスには `X-Inertia-Location` を持つ `409 Conflict` を使用する
明示的な location レスポンスには `X-Inertia-Version` を含めない

`WithPreserveFragment` は Inertia リクエストに対し、`X-Inertia-Redirect` を持つ `409 Conflict` を返す

## 部分リロード

`go-inertia` は文字列キーの map、slice、array を再帰的に解決する
ネストした `Optional`、`Defer`、`Merge`、`Once`、`Scroll` などの modifier には、`auth.notifications` のような完全なドット記法の metadata パスを使用する

- 絞り込みは `X-Inertia-Partial-Component` と描画するコンポーネントが一致する場合に適用する
- `X-Inertia-Partial-Data` は一致するパスと、そこへ到達するための祖先を含める
- `X-Inertia-Partial-Except` は一致するパスとその子孫を除外する
- `Partial-Except` だけがある場合は、optional と deferred を含め、除外されていない対象の props を解決する
- 両方のヘッダーがある場合は、`Partial-Data` を満たし、`Partial-Except` に一致しないパスを含める
- `X-Inertia-Reset` は指定した prop パスの merge metadata を削除する
  無限スクロールの場合は対応する `scrollProps` を保持し、reset を設定する
- `errors` は常に含める
- トップレベルの `flash` はデータがある場合に含め、prop の絞り込みから独立して扱う

通常の `func(*http.Request) (any, error)` props はレスポンスに含まれる場合に計算する
`Optional` は通常の訪問では省略し、一致する部分リロードの `Partial-Data` と `Partial-Except` で選択された場合に解決する
`Always` はフィルターの除外条件に一致する場合も含める

トップレベルのドット記法のキーは、再帰的な解決の前に展開する

```go
inertia.Props{
	"auth.user": currentUser,
	"auth.notifications": inertia.Defer(loadNotifications),
}
```

callback が map、slice、array を返す場合、その子要素は callback の値の一部として解決し、部分リロードのフィルターを再度適用しない

struct と独自の `json.Marshaler` は再帰処理を行わない JSON 値として扱う
ネストした modifier やフィールド単位の絞り込みが必要な場合は map、slice、array を使う
対応する値の構造と例は[部分リロード](partial-reloads_ja.md)を参照

## サーバーから提供する Head

`WithServerHead` は escape 済みの HTML 文字列を、設定した head prop と初回 HTML の `InertiaHead` に追加する
既定の prop 名は `head`

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

クライアントで `serverHead: true` を有効にするか、`Config.ServerHeadProp` と同じ独自の prop 名を指定する
詳細は[サーバーから提供する Head](server-head.md)を参照

## Deferred Props

`Defer` は初回ページオブジェクトから prop を省略し、その名前を `deferredProps` に追加する

```json
{
  "component": "Users/Index",
  "props": {
    "errors": {},
    "users": []
  },
  "url": "/users",
  "version": "",
  "deferredProps": {
    "default": ["permissions"]
  }
}
```

クライアントがコンポーネントの一致する部分リロードで deferred prop を要求すると、callback を評価し、解決した値を `props` に含める

## Once Props

`Once` は、クライアントが once key を取得済みと通知するまで通常どおり prop を解決する
クライアントは `X-Inertia-Except-Once-Props` で取得済みのキーを通知する

```json
{
  "component": "Dashboard",
  "props": {
    "errors": {},
    "plans": []
  },
  "url": "/dashboard",
  "version": "",
  "onceProps": {
    "plans": {
      "prop": "plans",
      "expiresAt": null
    }
  }
}
```

後続の Inertia リクエストに `X-Inertia-Except-Once-Props: plans` が含まれる場合、値を省略し、ページオブジェクトの `onceProps` metadata を保持する
コンポーネントが一致する部分リロードで選択された場合は、引き続き prop を解決する

## Merge Props

`Merge` は prop の値を含め、merge metadata をページオブジェクトへ追加する
通常の merge prop は prop のルートパスに追記する

```json
{
  "component": "Items/Index",
  "props": {
    "errors": {},
    "items": []
  },
  "url": "/items",
  "version": "",
  "mergeProps": ["items"]
}
```

ネストした append と prepend のパスは、ページ prop からの完全なパスで出力する

```json
{
  "mergeProps": ["results.data"],
  "prependProps": ["results.pinned"],
  "matchPropsOn": ["results.data.id"]
}
```

クライアントが `X-Inertia-Reset` を送った場合は、対応する merge metadata を省略するため、クライアントは値をマージせず置き換える

## Props の組み合わせ

`Defer`、`Merge`、`Once`、`Optional`、`Always`、computed props は共通の modifier モデルを使用する
deferred と merge、deferred と once、merge と once、optional と once などを組み合わせられる

```go
"results": inertia.Defer(loadResults).DeepMerge().MatchOn("data.id")
"permissions": inertia.Defer(loadPermissions).Once()
"activity": inertia.Merge(loadActivity).Once()
"companies": inertia.Optional(loadCompanies).Once()
```

## 無限スクロール

`Scroll` は prop の値と `scrollProps` metadata を追加し、部分リロード時に wrapper 内のデータを末尾または先頭へ追加するよう設定する

```json
{
  "component": "Posts/Index",
  "props": {
    "errors": {},
    "posts": {
      "data": []
    }
  },
  "url": "/posts?page=1",
  "version": "",
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

クライアントはスクロール用データを追加取得する際に `X-Inertia-Infinite-Scroll-Merge-Intent` を送る
値が `prepend` の場合、`go-inertia` は `mergeProps` の代わりに `prependProps` を出力する

```json
{
  "prependProps": ["posts.data"]
}
```

クライアントが `X-Inertia-Reset` も送った場合、merge と prepend の metadata を省略し、スクロールのエントリーに reset を設定する

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

## 履歴フラグ

`WithEncryptHistory` と `WithClearHistory` を使うと、ページオブジェクトに `encryptHistory` と `clearHistory` を設定できる

```go
return renderer.Render(w, req, "Account/Security", props,
	inertia.WithEncryptHistory(),
	inertia.WithClearHistory(),
)
```

## Prefetch リクエスト

Inertia クライアントは prefetch の訪問で `Purpose: prefetch` を送る
middleware や handler で prefetch による副作用を避ける必要がある場合は `IsPrefetch(req)` を使う

## Precognition

Precognition の検証リクエストは次のヘッダーを使用する:

- `Precognition: true`
- `Precognition-Validate-Only`: 選択したフィールドを検証する場合

`IsPrecognition(req)` と `PrecognitionValidateOnly(req)` でヘッダーを確認する

Precognition の検証成功時は `204 No Content` と次のヘッダーを返す:

- `Precognition: true`
- `Precognition-Success: true`
- `Vary: Precognition`

Precognition の検証失敗時は `422 Unprocessable Entity` と次の値を返す:

```json
{
  "errors": {
    "email": ["Email is required"]
  }
}
```

`PrecognitionSuccess` と `PrecognitionErrors` を使ってレスポンスを書き込む

## Deferred Props の Rescue

deferred の部分リロードで取得に失敗した際、prop を省略してキーを `rescuedProps` に含めるには `Defer(fn).Rescue()` を使う
`Rescue` がない場合は `Render` が取得時のエラーを返し、レスポンス本文は書き込まない

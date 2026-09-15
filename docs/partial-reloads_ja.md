# 部分リロードと Computed Props

[English](partial-reloads.md)

部分リロードを使うと、Inertia クライアントは同じページコンポーネントの props の一部を取得できる
`go-inertia` は標準の Inertia ヘッダーを読み取り、ネストした prop のパスを再帰的に絞り込む

## リクエストヘッダー

- `X-Inertia-Partial-Component` は描画するコンポーネント名と一致する必要がある
- `X-Inertia-Partial-Data` は取得する props を指定する
- `X-Inertia-Partial-Except` は除外する props とその子孫を指定する
- `except` だけの部分リロードでは、optional と deferred を含め、除外されていない対象の props を取得する
- 両方のフィルターがある場合は、`Partial-Data` を満たし、`Partial-Except` に一致しないパスを取得する
- `X-Inertia-Reset` は既存のクライアントデータへマージせず、置き換える merge または scroll props を指定する

`props.errors` は常に含まれる
トップレベルの `page.flash` は flash データがある場合に含まれ、ページ prop のフィルター対象にはならない

## ネストしたパス

ドット記法を使うと、同じ階層の他の値を解決せずに、ネストした一つの枝を取得できる

```ts
router.reload({ only: ['auth.notifications'] })
```

```go
inertia.Props{
	"auth": inertia.Props{
		"user": currentUser,
		"notifications": inertia.Defer(loadNotifications),
	},
}
```

トップレベルのキーにもドット記法を使用できる
prop の解決前に、ネストした map と連続するインデックスの配列へ展開される

```go
inertia.Props{
	"auth.user": currentUser,
	"users.0.name": "Ada",
}
```

配列のインデックスを含む部分パスは元のインデックスを保持する
絞り込み後に欠番が生じた場合、その配列は数値キーを持つ JSON オブジェクトとして出力される

ネストした値に prop modifier が必要な場合は、`inertia.Props`、文字列キーの map、slice、array を使う
struct と独自の `json.Marshaler` は再帰処理を行わない JSON 値として扱われる
通常の JSON データは格納できるが、内部の prop modifier の解決やフィールド単位の絞り込みは行われない

## Computed Props

通常の `func(*http.Request) (any, error)` prop は、そのパスがレスポンスに含まれる場合に評価される

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"users": users,
	"companies": func(req *http.Request) (any, error) {
		return loadCompanies(req.Context())
	},
})
```

通常の訪問では callback が実行される
コンポーネントが一致する部分リロードでは、適用中の `only` と `except` の条件をパスが満たす場合に実行される

明示的な wrapper の方が読みやすい場合は `Computed` を使う

```go
"companies": inertia.Computed(loadCompanies)
```

`Lazy` は `Computed` の非推奨 alias であり、Inertia v3 で削除された `LazyProp` とは異なる
部分リロードで選択された場合に返す props には `Optional` を使う

## Optional Props

`Optional` は通常の訪問では省略され、同じコンポーネントへの部分リロードで選択された場合に解決される

```go
"companies": inertia.Optional(loadCompanies)
```

optional prop の `companies` と通常の prop の `users` がある場合の例:

| 訪問 | `companies` を解決するか |
| --- | --- |
| 通常の訪問 | しない |
| `only: ['companies']` | する |
| `only: ['users']` | しない |
| `except: ['users']` | する |
| `except: ['companies']` | しない |
| `only: ['companies'], except: ['companies']` | しない |

部分リロードの行はコンポーネントが一致し、どちらの prop にも追加の modifier がない場合を示す
`except` によって、まだ取得していない optional データも読み込まれることがある
特定の optional props だけを取得したい場合は `only` を使う

## Always Props

部分リロードでも返したい props には `Always` を使う

```go
"auth": inertia.Always(func(req *http.Request) (any, error) {
	return currentUser(req.Context())
})
```

`Always` は現在のユーザーデータや機能フラグなど、ページ全体で最新に保ちたい値に適している

## 組み合わせ

computed、optional、always、deferred、merge、once の各 modifier は共通の prop モデルを使用する

```go
"companies": inertia.Optional(loadCompanies).Once()
"results": inertia.Defer(loadResults).DeepMerge().MatchOn("data.id")
```

## Modifier の検証

`Render` は部分リロードの絞り込み前に modifier の組み合わせを検証する
プロトコル上の意味を持たない組み合わせには `ErrInvalidPropConfiguration` に一致するエラーを返す
`PropConfigurationError.Path` はネストした prop の完全なパスを保持する

`Defer` のない `Rescue`、merge または scroll のない `MatchOn`、scroll のない `Wrapper`、単一値を取る可変長引数への余分な指定などはエラーになる

要求されていない props も検証するため、設定ミスが次回の部分リロードまで見つからない状態を防げる

## 参照

サーバー側の props 評価ルールは [Inertia プロトコル](https://inertiajs.com/docs/v3/core-concepts/the-protocol)を参照

# Once Props

[English](once-props.md)

Once props は Inertia クライアントが一度受け取ると再利用できる props
料金プラン、機能フラグ、権限一覧など、取得コストが高く変更の少ないデータに適している

## サーバーでの使用

prop を `Once` で包む

```go
err := renderer.Render(w, req, "Dashboard", inertia.Props{
	"plans": inertia.Once(func(req *http.Request) (any, error) {
		return loadPlans(req.Context())
	}),
})
```

クライアントが once key を取得済みと通知するまで callback が実行される
その後は prop の値を省略し、ページオブジェクトに `onceProps` metadata を残す

## 独自のキー

既定の once key は prop 名
ページや prop 名をまたいで同じキーを共有する場合は `As` を使う

```go
err := renderer.Render(w, req, "Users/Index", inertia.Props{
	"availableRoles": inertia.Once(loadRoles).As("roles"),
})
```

ページオブジェクトには次の値が含まれる:

```json
{
  "onceProps": {
    "roles": {
      "prop": "availableRoles",
      "expiresAt": null
    }
  }
}
```

## 強制的な再取得

クライアントが once key を取得済みと通知した場合も値を解決するには `Fresh` を使う

```go
"plans": inertia.Once(loadPlans).Fresh()
```

条件に応じて再取得を無効にする場合は `false` を渡す

```go
"plans": inertia.Once(loadPlans).Fresh(forceRefresh)
```

## 有効期限

有効期限のタイムスタンプをクライアントへ送るには `Until` を使う

```go
"plans": inertia.Once(loadPlans).Until(time.Now().Add(30 * 24 * time.Hour))
```

タイムスタンプは Unix ミリ秒で `onceProps` の `expiresAt` に出力される
`Until` を指定しない場合、`expiresAt` は `null`

## Modifier の組み合わせ

Once props は deferred、merge、optional、computed props と組み合わせられる

```go
"permissions": inertia.Defer(loadPermissions).Once()
"activity": inertia.Merge(loadActivity).Once()
"companies": inertia.Optional(loadCompanies).Once()
```

deferred と optional props は初回レスポンスでは値を省略し、`onceProps` metadata を送る
コンポーネントが一致する部分リロードでは、後述の `only` と `except` の条件を満たす値が返される

## 部分リロード時の挙動

コンポーネントが一致する部分リロードでは、`X-Inertia-Partial-Data` がある場合はその条件を満たし、`X-Inertia-Partial-Except` に一致しない once prop を解決する
`except` だけの部分リロードでも、除外されていない once props は解決される
両方のヘッダーに同じ prop が指定された場合、その prop は除外される
クライアントが `X-Inertia-Except-Once-Props` にそのキーを指定してもこのルールが適用されるため、部分リロードで取得済みデータを更新できる

コンポーネントが一致する部分リロードで once prop が選択されない場合、callback は実行されず、once metadata もそのレスポンスから省略される

optional props を含むフィルターの例は[部分リロード](partial-reloads_ja.md)を参照

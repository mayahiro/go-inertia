# DevTools

`go-inertia`はbackend-independentな
[Inertia DevTools protocol](https://inertiajs.com/docs/v3/advanced/devtools-protocol)
を実装し、runtime dependencyを追加せずにInertia DevTools Chrome extension向けのrequest、response、page、prop、route、source metadataを記録する

DevToolsは既定で無効であり、明示的な有効化が必要となる

## セットアップ

Rendererの構築時にrecorderを有効化する

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

library自体は`INERTIA_DEVTOOLS_ENABLED`を読み取らない
上の環境変数は公式adapterと同梱のEcho exampleに合わせたapplication側の規約となる

requestの観測とread APIの提供には`Renderer.Middleware`または対応するframework adapter middlewareを登録する

`createInertiaApp`でclient hookを有効化する

```ts
createInertiaApp({
  // ...
  dev: import.meta.env.DEV,
})
```

client adapterはInertia DevTools対応版である必要があり、公式Inertia documentationでは3.6以降が必要とされている
[Inertia DevTools Chrome extension](https://inertiajs.com/docs/v3/advanced/devtools)
をinstallし、applicationの利用中にInertia panelを開く

## 設定

| option | 既定値 | 挙動 |
| --- | --- | --- |
| `Enabled` | `false` | recording、discovery header、初回HTMLのdiscovery tag、read APIを有効化する |
| `Authorize` | direct loopback requestのみ | recordingとread API accessの両方を制御する |
| `Limit` | `100` | browser tabごとに保持するentry数の上限となる |
| `TTL` | `24h` | 後続のstore操作時に古いentryを削除する |
| `MaxBodyBytes` | `256000` | captureするrequest bodyを制限する |
| `RedactKeys` | 空 | 組み込みsecret listへ大文字小文字を区別しないJSON keyとform keyを追加する |
| `RedactHeaders` | 空 | 組み込みsecret listへ大文字小文字を区別しないheaderを追加する |
| `ComponentPathResolver` | `nil` | component名をfrontend source pathへ対応付ける |

`Limit`、`TTL`、`MaxBodyBytes`へ1未満の値を指定した場合は既定値を使う

in-memory storeは1つの`Renderer` instanceに属し、processの再起動で消去される
複数process間ではentryを共有しないため、read requestはresponseを記録したprocessへ到達する必要がある

## セキュリティ

DevTools entryにはpage props、header、request dataが含まれる可能性があるため、信頼できる開発環境でのみrecorderを有効化する

`Authorize`がnilの場合、recordingとreadの両方で`Request.RemoteAddr`のdirect peerがloopback addressであることを要求する
forwarded headerは意図的に信頼しない
reverse proxyにより外部requestがproxy自身のloopback接続から来たように見える場合があるため、有効化したapplicationへproxy経由で到達できる環境では既定のgateだけに依存しない
その場合はapplicationのdeveloper認証に基づく`Authorize` callbackを指定する

recorderは保存前にpassword、token、secret、client secret、API keyを常にredactする
cookie、authorization header、proxy authorization header、XSRF token、CSRF tokenも常にredactする
`RedactKeys`と`RedactHeaders`は既定値へ対象を追加し、組み込みの保護対象を削除しない
記録するpage、request、redirect URL内でkeyが一致するquery parameterもredactする

read API responseには`Cache-Control: no-store`と`X-Content-Type-Options: nosniff`を設定する
recorderまたはresolverの失敗はapplication responseから分離する

## 記録データと制限

認可された各requestには`X-Inertia-Devtools-Id`と`X-Inertia-Devtools-Parent-Out`を追加する
初回Inertia HTMLには必須の`script[data-inertia-devtools-id]` discovery elementも追加する
clientから届く相関用request headerはentry metadataへ保持する

Inertia page responseにはredact済みpage object、解決済みprop value、prop modifier metadata、設定されている場合はcomponent source pathを含める
Echo adapterはmatched route URIと`Adapter.Render`または`Adapter.RenderError`を呼んだapplication側の位置も記録する

raw non-Inertia response bodyは意図的にbufferせず、omittedとして記録する
statusとheaderは引き続き記録する
request bodyはdownstreamのapplication codeが最後まで読み取った場合にcaptureし、一部だけ読み取られたbodyはomittedとして記録する
JSONとformの値はredactし、multipart fileはname、size、media typeだけへ縮約し、non-textualまたは`MaxBodyBytes`を超えるbodyは省略する

framework adapterは次のno-op safeなhookでmetadataを追加できる

```go
renderer.RecordDevToolsRoute(req, name, uri, action)
renderer.RecordDevToolsRenderSource(req, file, line)
```

どちらのmethodもrecorderが受け付けていないrequestでは何もしない

## Read API

protocol pathは固定となる

- `GET /_inertia/devtools/entries/{id}`は1件のentryまたは`404`を返す
- `GET /_inertia/devtools/entries`は新しい順にentryを返す

DevToolsが有効な間、`Renderer.Middleware`はこれらのpathを予約し、downstreamのapplication handlerより先に処理する

list endpointは`component`、`type`、`exclude`、`offset`、`limit`のquery filterに対応する
その他のmethodは`405`を返す

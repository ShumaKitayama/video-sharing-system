# デプロイ構成とストレージ設計（Vercel 本番運用）

このドキュメントは、当初の設計書（`README.md` / `api-design.md` / `database-design.md`）に無かった、**本番運用（Vercel）に必要な追加設計**をまとめる正本である。

対象は次の 3 点である。

1. Vercel での本番デプロイ構成（フロント / API の 2 プロジェクト）
2. 動画ファイルの永続化ストレージ（ローカルディスク ↔ Cloudflare R2 の二層構成）
3. 大容量アップロードのための「ブラウザ → ストレージ直接アップロード」方式

> ⚠️ この機能群は「教室 LAN 前提」という基本方針の**例外**として、インターネット上（Vercel）でも動かせるようにするための最小限の追加である。ローカル LAN・ネイティブ実行の動作は一切壊さない（デュアルモードを維持する）。

---

## 0. 変更履歴: Vercel Blob から Cloudflare R2 へ

当初は動画実体を **Vercel Blob** に保存していたが、無料枠（保存 1GB / 転送 10GB per 月）に達したため、**Cloudflare R2** へ移行した。

| | Vercel Blob (旧) | Cloudflare R2 (現行) |
| --- | --- | --- |
| 無料の保存容量 | 1GB | 10GB |
| 無料の転送量 | 10GB/月 | **無制限（egress 無料）** |
| 認証方式 | 独自トークン | S3 互換（AWS SigV4） |

動画配信は転送量が支配的なので、egress が無料である点が決め手である。

**移行前にアップロードされた動画はそのまま再生できる。** DB の `storage_key` にはフル URL が入っており、配信は「URL ならリダイレクト」で処理されるため、保存先が混在していても問題ない。旧動画の削除だけは Vercel Blob のトークンを必要とするので、`BLOB_READ_WRITE_TOKEN` は残しておく。

---

## 1. 全体像

### 1.1 ローカル開発（従来どおり）

```txt
ブラウザ (localhost:5173)
   │  Cookie: session_id
   ├─ JSON / multipart ─────────────▶ Go API (localhost:8080)
   │                                     ├─ PostgreSQL（メタデータ）
   │                                     └─ ローカルディスク UPLOAD_DIR（動画実体）
   └─ 動画再生 GET .../stream ────────▶ ディスクから Range 配信（200 / 206）
```

### 1.2 本番（Vercel）

```txt
ブラウザ
   │
   │  同一オリジン（フロントの rewrite プロキシ経由）
   ├─ 小さな JSON / 認証 ─────────────▶ フロント(静的) ──rewrite /api/*──▶ Go API(コンテナ)
   │                                                                        ├─ Neon PostgreSQL
   │                                                                        └─ Cloudflare R2(動画実体)
   │
   ├─ 動画本体アップロード ───────────────────────────────▶ R2（署名付きURLへ直接 PUT・API を経由しない）
   │
   └─ 動画再生 GET .../stream ─▶ Go API が 307 リダイレクト ─▶ R2 の公開 URL
```

| 役割 | Vercel プロジェクト | 本番 URL |
| --- | --- | --- |
| フロント（静的 + rewrite） | `video-sharing-front` | `https://video-sharing-front.vercel.app` |
| API（Go コンテナ） | `video-sharing-api` | `https://video-sharing-api.vercel.app` |

- **DB**: Neon（マネージド PostgreSQL）。`DATABASE_URL` で接続。
- **動画実体**: Cloudflare R2 バケット（公開読み取り）。`R2_*` 環境変数で認証。

---

## 2. なぜこの構成が必要か（背景）

同じ「フロントとバックエンド」でも、ローカル LAN と Vercel では制約が違う。次の 3 つの壁が原因で、素朴な構成のままでは本番で動かなかった。

### 2.1 壁 A: サードパーティ Cookie のブロック

フロントと API が別ドメインだと、ログイン Cookie（`session_id`）が「サードパーティ Cookie」となりブラウザにブロックされる。

**対策**: フロント側 `vercel.json` の rewrite で `/api/*` を API に中継し、**同一オリジン化**する。ブラウザから見ると API はフロントと同じドメインになり、Cookie が正しく送られる。

```jsonc
// front/vercel.json（イメージ）
{
  "rewrites": [
    { "source": "/api/:path*", "destination": "https://video-sharing-api.vercel.app/api/:path*" }
  ]
}
```

これにより、フロントの API 呼び出しは相対パス（`API_BASE_URL = ""`）で行う。

### 2.2 壁 B: `/tmp` はインスタンス間で共有されない

Vercel のコンテナ/関数はリクエストごとに別インスタンスになり得る。ローカルディスク（`/tmp`）に保存した動画は、次のリクエストでは別インスタンスに繋がって**消えている**ように見え、再生が 404 になる。

**対策**: 動画実体を **Cloudflare R2**（永続オブジェクトストレージ）に保存する。DB の `storage_key` には公開 URL（`https://pub-xxxx.r2.dev/...`）を保存する。

### 2.3 壁 C: リクエストボディの約 4.5MB 上限

Vercel の関数/コンテナには**リクエストボディ約 4.5MB の上限**がある。動画本体を `POST /videos`（multipart）で API に送ると、Gin に届く前にプラットフォーム層で `413` が返る。この 413 応答には CORS ヘッダが付かないため、ブラウザ上は「CORS エラー」に見える（CORS 設定の問題ではない）。

**対策**: 動画本体を API に通さず、**ブラウザから R2 へ直接アップロード**する（第 4 章）。API にはアップロード完了後の小さな JSON だけを送る。

---

## 3. ストレージ二層構成

動画実体の保存先は環境変数で自動的に切り替わる。コードの分岐は `storage` 層に閉じ込め、上位層（service / handler）は保存先を意識しない。

| 環境 | 判定 | 保存先 | `storage_key` の中身 |
| --- | --- | --- | --- |
| ローカル / Docker | `R2_*` 未設定 | ローカルディスク `UPLOAD_DIR` | 相対キー `videos/2026/08/<uuid>.mp4` |
| Vercel 本番 | `R2_*` 設定あり | Cloudflare R2 | 公開 URL `https://pub-xxxx.r2.dev/<uuid>.mp4` |
| 移行前の旧データ | — | Vercel Blob（そのまま） | 公開 URL `https://<store>.public.blob.vercel-storage.com/<uuid>.mp4` |

`R2_*` は **5 つ全部を設定するか、1 つも設定しないか**のどちらかにする。中途半端に設定した場合はサーバー起動時にエラーで停止する（タイプミスが「ストレージ未設定」に化けるのを防ぐため）。

### 3.1 関連ファイル

| ファイル | 役割 |
| --- | --- |
| `server/internal/storage/r2.go` | R2 との通信すべて。AWS SigV4 署名、署名付きURL生成、PUT / DELETE |
| `server/internal/storage/r2_test.go` | 署名が AWS 公式実装（botocore）と一致することを検証する契約テスト |
| `server/internal/storage/object.go` | 公開 URL への HEAD（実体確認） |
| `server/internal/storage/blob.go` | 旧 Vercel Blob データの削除のみ |
| `server/internal/storage/video.go` | 保存先の切り替え、`storage_key` 生成、削除の振り分け |
| `server/internal/streaming/serve.go` | `storage_key` が URL なら 307 リダイレクト、そうでなければ Range 配信 |

### 3.2 配信の分岐（307 リダイレクト）

`GET /api/v1/videos/:id/stream` は `storage_key` の形で動きを変える。

```txt
storage_key が "https://..." で始まる  → 307 Temporary Redirect（公開 URL へ）
それ以外（相対キー）                    → ディスクから http.ServeContent で Range 配信
```

本番のストリームログに `307` が出るのは正常動作である（ブラウザがストレージの URL へ辿って再生する）。R2 は Range リクエストに対応しているので、シークも問題なく動く。

### 3.3 削除の振り分け

`storage_key` の形を見て、そのファイルを持っている provider に処理を渡す。

```txt
R2 の公開ベースURLと一致        → R2 に署名付き DELETE
*.blob.vercel-storage.com       → Vercel Blob の削除 API（BLOB_READ_WRITE_TOKEN が必要）
相対キー                        → ローカルディスクから os.Remove
```

---

## 4. 大容量アップロード（ブラウザ → R2 直接アップロード）

壁 C を回避するため、**署名付きURL（presigned URL）方式**を採用する。**動画本体は一度も API を経由しない**。

### 4.1 シーケンス

```txt
① ブラウザ ──POST /api/v1/uploads/token（Cookie 認証・同一オリジン）──▶ API
   ◀── 短命アップロードトークン（HMAC 署名・15分）

② ブラウザ ──POST /api/v1/uploads/direct { content_type }────────────▶ API
   （X-Upload-Token に ① のトークン。API が署名付きURLを発行）
   ◀── { upload_url, public_url, content_type, expires_in }

③ ブラウザ ──PUT 動画本体───────────────────────────▶ R2（直接・4.5MB制限の対象外）
   ◀── 200

④ ブラウザ ──POST /api/v1/videos（JSON, 小さい）──────────▶ API
   { blob_url: public_url, title, description, content_type, duration_seconds }
   API が HEAD で実体を確認し DB 登録
   ◀── 201 { id, title, status }
```

- ①②④ はどれも小さいリクエストなので、同一オリジン（rewrite 経由）で送っても 4.5MB 制限に当たらない。
- ③ だけが大きく、これは R2 へ直接送るので制限を回避できる。
- ④ のフィールド名が `blob_url` のままなのは、旧方式との互換のため。中身は「アップロード先の公開 URL」である。

### 4.2 追加エンドポイント

| メソッド | パス | 認証 | 役割 |
| --- | --- | --- | --- |
| POST | `/api/v1/uploads/token` | ログイン必須（Cookie） | 短命アップロードトークンを発行 |
| POST | `/api/v1/uploads/direct` | Cookie または上記トークン | 署名付きアップロードURLを発行 |

`POST /api/v1/videos` は **2 つの入力形式**を持つ（Content-Type で分岐）。

| Content-Type | 意味 | 主な利用 |
| --- | --- | --- |
| `multipart/form-data` | 動画本体を含む従来方式 | ローカル開発 |
| `application/json` | アップロード済み URL の登録のみ | 本番（直接アップロード後） |

### 4.3 署名付きURLの安全性

発行される URL は次のように限定されており、Cloudflare の認証情報がブラウザに渡ることはない。

| 制限 | 内容 |
| --- | --- |
| オブジェクトキー | サーバーが決めた UUID 1 個のみ。他のファイルを上書きできない |
| Content-Type | 署名に含まれるため、MP4 用の枠で別形式は保存できない |
| 有効期限 | 30 分 |
| 操作 | PUT のみ。読み出しや一覧取得はできない |

サイズだけは署名で縛れないため、④ の登録時に HEAD でサイズを確認し、500MB を超えていたらそのオブジェクトを削除して `413` を返す。

### 4.4 SigV4 署名の実装（Go）

R2 は S3 互換 API なので、署名は AWS Signature Version 4 に従う。Node の SDK を持ち込まず、Go の標準ライブラリだけで実装している（`server/internal/storage/r2.go`）。

```txt
① 正規化リクエストを組み立てる
   METHOD / パス / クエリ / ヘッダ / UNSIGNED-PAYLOAD
② ①のSHA256を「署名対象文字列」に埋め込む
   AWS4-HMAC-SHA256 / 日時 / スコープ(日付/auto/s3/aws4_request) / hash
③ シークレットから HMAC を4回連鎖させて署名鍵を導出する
   "AWS4"+secret → 日付 → リージョン(auto) → サービス(s3) → aws4_request
④ ②を③で HMAC-SHA256 し、X-Amz-Signature としてクエリに付ける
```

1 バイトでも違うと R2 に拒否されるため、`r2_test.go` で **AWS 公式実装（botocore）が生成した URL と完全一致すること**を検証している。

実際のバケットに対する疎通確認（発行 → PUT → HEAD → 削除）は `r2_live_test.go` で行える。環境変数が無いときは自動でスキップされる。

```bash
cd server
R2_ACCOUNT_ID=... R2_ACCESS_KEY_ID=... R2_SECRET_ACCESS_KEY=... \
R2_BUCKET=... R2_PUBLIC_BASE_URL=... \
go test ./internal/storage/ -run TestLiveR2RoundTrip -v
```

### 4.5 フロント実装

| ファイル | 役割 |
| --- | --- |
| `front/src/hooks/useVideoUpload.ts` | 本番は署名付きURLへ直接 PUT、ローカルは multipart。進捗付き |
| `front/src/api/endpoints.ts` | `uploads.token` / `uploads.direct` / `API_DIRECT_BASE_URL` |

外部依存は無い（`XMLHttpRequest` のみ）。旧方式で使っていた `@vercel/blob` パッケージは削除済み。

分岐の考え方:

- `import.meta.env.PROD` または `VITE_REMOTE_BACKEND=true`: ①〜④ の直接アップロード方式。
- それ以外（ローカル開発）: 従来どおり `multipart/form-data` を API に直接送る。

---

## 5. 環境変数一覧

### 5.1 API（`video-sharing-api`）

| 変数 | 必須 | 説明 |
| --- | --- | --- |
| `DATABASE_URL` | ✅ | PostgreSQL 接続文字列（本番は Neon、`sslmode=require`） |
| `R2_ACCOUNT_ID` | 本番✅ | Cloudflare のアカウントID |
| `R2_ACCESS_KEY_ID` | 本番✅ | R2 API トークンのアクセスキーID |
| `R2_SECRET_ACCESS_KEY` | 本番✅ | R2 API トークンのシークレット |
| `R2_BUCKET` | 本番✅ | バケット名 |
| `R2_PUBLIC_BASE_URL` | 本番✅ | 公開読み取り用のベースURL（`https://pub-xxxx.r2.dev`） |
| `BLOB_READ_WRITE_TOKEN` | 任意 | 移行前の旧動画を削除するためだけに必要 |
| `CORS_ORIGINS` | 任意 | 許可するフロントオリジン（カンマ区切り） |
| `COOKIE_SECURE` | 任意 | 本番 HTTPS では `true` |
| `COOKIE_SAMESITE` | 任意 | 既定 `lax`（同一オリジン化しているため `lax` で可） |
| `UPLOAD_TOKEN_SECRET` | 任意 | アップロードトークンの署名鍵。未設定なら `DATABASE_URL` から導出 |
| `UPLOAD_DIR` / `MIGRATIONS_DIR` / `PORT` | 任意 | ローカル/コンテナのパスとポート |

### 5.2 フロント（`video-sharing-front`）

| 変数 | 必須 | 説明 |
| --- | --- | --- |
| `VITE_API_BASE_URL` | 任意 | 通常は**未設定**（本番は rewrite により相対パス `""`） |
| `VITE_API_DIRECT_URL` | 任意 | ローカルの multipart 直接送信先。既定は `http://localhost:8080` |

> 署名付きURLは API 側で発行するため、**フロントに R2 の認証情報は不要**。

---

## 6. Cloudflare R2 側の設定

### 6.1 バケット

1. Cloudflare ダッシュボード → **R2 Object Storage** → **Create bucket**
2. バケット名（例: `video-sharing-uploads`）、ロケーションは `Automatic`
3. 作成後、**Settings → Public Development URL** を **Enable** にする  
   → 表示される `https://pub-xxxxxxxx.r2.dev` が `R2_PUBLIC_BASE_URL`

### 6.2 CORS（これが無いとブラウザからの PUT が失敗する）

**Settings → CORS Policy → Edit** に次を設定する。`AllowedOrigins` は自分のフロントの URL に置き換える。

```json
[
  {
    "AllowedOrigins": [
      "https://video-sharing-front.vercel.app",
      "http://localhost:5173"
    ],
    "AllowedMethods": ["PUT", "GET", "HEAD"],
    "AllowedHeaders": ["content-type"],
    "MaxAgeSeconds": 3600
  }
]
```

### 6.3 API トークン

**R2 Object Storage → API → Manage API Tokens → Create API Token**

- Permissions: **Object Read & Write**
- Specify bucket: 作成したバケットのみ
- 発行後に表示される **Access Key ID / Secret Access Key** を控える（再表示不可）

`Account ID` は R2 の概要ページ、または S3 API エンドポイント `https://<ACCOUNT_ID>.r2.cloudflarestorage.com` から読み取る。

---

## 7. デプロイ手順

### 7.0 モノレポの Root Directory（Git 連携で必須）

リポジトリは `server/`（API）と `front/`（フロント）のモノレポ構成である。Vercel プロジェクトごとに **Root Directory** を設定しないと、PR の Git 連携デプロイがリポジトリルート（`.`）で実行され、コンテナビルドが 14ms で空終了 → **Error** になる。

| Vercel プロジェクト | Root Directory | Framework |
| --- | --- | --- |
| `video-sharing-api` | `server` | Container（`Dockerfile.vercel`） |
| `video-sharing-front` | `front` | Vite |

確認コマンド:

```bash
npx vercel@latest project inspect video-sharing-api   # Root Directory: server
npx vercel@latest project inspect video-sharing-front # Root Directory: front
```

Dashboard: Project Settings → General → **Root Directory**。

### 7.1 デプロイ

環境変数（`R2_*`）を **先に** Vercel に登録してからデプロイする。順番を逆にすると、一時的にアップロードが `503 動画の保存先が設定されていません` になる。

```bash
# API（Go コンテナ）
cd server
npx vercel@latest --prod

# フロント（静的 + rewrite）
cd ../front
npx vercel@latest --prod
```

### 7.2 動作確認（本番）

1. `https://video-sharing-front.vercel.app` をスーパーリロード（キャッシュ削除）
2. ログイン → 動画アップロード
3. ブラウザ Network タブで想定の流れ:
   - `POST /api/v1/uploads/token` → `200`
   - `POST /api/v1/uploads/direct` → `200`
   - `PUT https://….r2.cloudflarestorage.com/…` → `200`（本体・直接）
   - `POST /api/v1/videos` → `201`
   - 再生時 `GET .../stream` → `307`（R2 へリダイレクト）

---

## 8. 既知の注意点

- **Blob 導入前にアップロードした動画**は DB に残るがファイル実体が無い（`/tmp` 消失）。再アップロードするか削除する。
- **R2 移行前の動画**は Vercel Blob に残っており、そのまま再生できる。削除機能を効かせたい場合は `BLOB_READ_WRITE_TOKEN` を残したままにする。
- `PUT` が `403` で失敗する場合、まず R2 の **CORS Policy** と **API トークンの権限（Object Read & Write）** を疑う。
- サーバー起動ログに `video storage: Cloudflare R2 bucket "..."` と出ていれば R2 が有効。`local disk` と出ていたら環境変数が読めていない。
- ローカル開発は R2 を使わない。`R2_*` をローカルに設定すると本番バケットを汚すため、通常は設定しない。
- **PR の Vercel チェックが Error（Build 14ms で終了）** → Root Directory が `.` のままになっていないか確認（上記 7.0）。

---

## 9. このドキュメントと他資料の関係

| 資料 | 関係 |
| --- | --- |
| [`api-design.md`](./api-design.md) | エンドポイント仕様の正本。本書はアップロード系の追加分を補足 |
| [`database-design.md`](./database-design.md) | `storage_key` は相対キーまたは公開 URL のいずれか |
| [`BACKEND_API_GUIDE.md`](./BACKEND_API_GUIDE.md) | フロントからの呼び出し手順。本番の直接アップロードも記載 |

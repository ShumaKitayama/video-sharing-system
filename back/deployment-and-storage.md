# デプロイ構成とストレージ設計（Vercel 本番運用）

このドキュメントは、当初の設計書（`README.md` / `api-design.md` / `database-design.md`）に無かった、**本番運用（Vercel）に必要な追加設計**をまとめる正本である。

対象は次の 3 点である。

1. Vercel での本番デプロイ構成（フロント / API の 2 プロジェクト）
2. 動画ファイルの永続化ストレージ（ローカルディスク ↔ Vercel Blob の二層構成）
3. 大容量アップロードのための「ブラウザ → Blob 直接アップロード」方式

> ⚠️ この機能群は「教室 LAN 前提」という基本方針の**例外**として、インターネット上（Vercel）でも動かせるようにするための最小限の追加である。ローカル LAN・ネイティブ実行の動作は一切壊さない（デュアルモードを維持する）。

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
   │                                                                        └─ Vercel Blob(動画実体)
   │
   ├─ 動画本体アップロード ───────────────────────────────▶ Vercel Blob（直接 PUT・API を経由しない）
   │
   └─ 動画再生 GET .../stream ─▶ Go API が 307 リダイレクト ─▶ Vercel Blob の公開 URL
```

| 役割 | Vercel プロジェクト | 本番 URL |
| --- | --- | --- |
| フロント（静的 + rewrite） | `video-sharing-front` | `https://video-sharing-front.vercel.app` |
| API（Go コンテナ） | `video-sharing-api` | `https://video-sharing-api.vercel.app` |

- **DB**: Neon（マネージド PostgreSQL）。`DATABASE_URL` で接続。
- **動画実体**: Vercel Blob ストア `video-sharing-uploads`（public）。`BLOB_READ_WRITE_TOKEN` で認証。

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

**対策**: 動画実体を **Vercel Blob**（永続オブジェクトストレージ）に保存する。DB の `storage_key` には Blob の公開 URL（`https://...blob.vercel-storage.com/...`）を保存する。

### 2.3 壁 C: リクエストボディの約 4.5MB 上限

Vercel の関数/コンテナには**リクエストボディ約 4.5MB の上限**がある。動画本体を `POST /videos`（multipart）で API に送ると、Gin に届く前にプラットフォーム層で `413` が返る。この 413 応答には CORS ヘッダが付かないため、ブラウザ上は「CORS エラー」に見える（CORS 設定の問題ではない）。

**対策**: 動画本体を API に通さず、**ブラウザから Vercel Blob へ直接アップロード**する（第 4 章）。API にはアップロード完了後の小さな JSON だけを送る。

---

## 3. ストレージ二層構成

動画実体の保存先は環境変数で自動的に切り替わる。コードの分岐は `storage` 層に閉じ込め、上位層（service / handler）は保存先を意識しない。

| 環境 | 判定 | 保存先 | `storage_key` の中身 |
| --- | --- | --- | --- |
| ローカル / Docker | `BLOB_READ_WRITE_TOKEN` 未設定 | ローカルディスク `UPLOAD_DIR` | 相対キー `videos/2026/07/<uuid>.mp4` |
| Vercel 本番 | `BLOB_READ_WRITE_TOKEN` 設定あり | Vercel Blob | 公開 URL `https://<store>.public.blob.vercel-storage.com/<uuid>.mp4` |

### 3.1 関連ファイル

| ファイル | 役割 |
| --- | --- |
| `server/internal/storage/blob.go` | Blob の PUT / DELETE / HEAD、クライアントトークン生成 |
| `server/internal/storage/video.go` | 保存先の切り替え（ディスク / Blob）、`storage_key` 生成 |
| `server/internal/streaming/serve.go` | `storage_key` が URL なら 307 リダイレクト、そうでなければ Range 配信 |

### 3.2 配信の分岐（307 リダイレクト）

`GET /api/v1/videos/:id/stream` は `storage_key` の形で動きを変える。

```txt
storage_key が "https://..." で始まる  → 307 Temporary Redirect（Blob の公開 URL へ）
それ以外（相対キー）                    → ディスクから http.ServeContent で Range 配信
```

本番のストリームログに `307` が出るのは正常動作である（ブラウザが Blob の URL へ辿って再生する）。

---

## 4. 大容量アップロード（ブラウザ → Blob 直接アップロード）

壁 C を回避するため、`@vercel/blob` の「クライアントアップロード」方式を採用する。**動画本体は一度も API を経由しない**。

### 4.1 シーケンス

```txt
① ブラウザ ──POST /api/v1/uploads/token（Cookie 認証・同一オリジン）──▶ API
   ◀── 短命アップロードトークン（HMAC 署名・15分）

② ブラウザ ──POST /api/v1/uploads/blob（@vercel/blob プロトコル）────▶ API
   （clientPayload に ① のトークン。API が Blob クライアントトークンを発行）
   ◀── clientToken（vercel_blob_client_...）

③ ブラウザ ──PUT 動画本体────────────────────────────▶ Vercel Blob（直接・4.5MB制限の対象外）
   ◀── アップロード完了・公開 URL

④ ブラウザ ──POST /api/v1/videos（JSON, 小さい）──────────▶ API
   { blob_url, title, description, content_type, duration_seconds }
   API が HEAD で実体を確認し DB 登録
   ◀── 201 { id, title, status }
```

- ①②④ はどれも小さいリクエストなので、同一オリジン（rewrite 経由）で送っても 4.5MB 制限に当たらない。
- ③ だけが大きく、これは Vercel Blob へ直接送るので制限を回避できる。

### 4.2 追加エンドポイント

| メソッド | パス | 認証 | 役割 |
| --- | --- | --- | --- |
| POST | `/api/v1/uploads/token` | ログイン必須（Cookie） | 短命アップロードトークンを発行 |
| POST | `/api/v1/uploads/blob` | Cookie または上記トークン | `@vercel/blob` クライアントアップロードのトークン発行窓口 |

`POST /api/v1/videos` は **2 つの入力形式**を持つ（Content-Type で分岐）。

| Content-Type | 意味 | 主な利用 |
| --- | --- | --- |
| `multipart/form-data` | 動画本体を含む従来方式 | ローカル開発 |
| `application/json` | Blob 済み URL の登録のみ | 本番（Blob 直接アップロード後） |

`application/json` 版のボディ:

```json
{
  "blob_url": "https://<store>.public.blob.vercel-storage.com/<uuid>.mp4",
  "title": "理科の実験",
  "description": "水の状態変化",
  "content_type": "video/mp4",
  "duration_seconds": 42
}
```

サーバーは受け取った `blob_url` をそのまま信用せず、次を検証する。

1. ホスト名が `*.blob.vercel-storage.com` であること
2. HEAD リクエストで実体が存在し、サイズ・Content-Type が妥当であること（500MB 以下、MP4 / WebM）

### 4.3 アップロードトークン（HMAC）

`POST /uploads/token` が返すトークンは、`user_id:role:exp:HMAC` 形式の署名付き文字列である。

- 署名鍵は `UPLOAD_TOKEN_SECRET`。未設定時は `DATABASE_URL` から決定的に導出する（開発を止めないためのフォールバック）。
- 有効期限は 15 分。
- 用途は「Blob クライアントトークン発行の認可」と「Blob 直後の登録リクエストの補助認証」。

関連ファイル:

| ファイル | 役割 |
| --- | --- |
| `server/internal/service/upload_token.go` | トークン発行 / 検証 |
| `server/internal/middleware/upload_auth.go` | Cookie またはトークンでの認可 |
| `server/internal/api/v1/upload_handlers.go` | `/uploads/token`, `/uploads/blob` ハンドラ |

### 4.4 Blob クライアントトークンの生成（Go 実装）

`/uploads/blob` は `@vercel/blob` クライアントの `handleUploadUrl` プロトコルを Go で実装している。Node 関数を追加せず、API（Go）だけで完結させるための選択である。

トークン書式は `@vercel/blob` と同一にする必要がある。

```txt
vercel_blob_client_<storeId>_<base64( hmacHex + "." + base64(payload) )>

storeId    = READ/WRITE トークンを "_" で分割した 4 番目の要素
payload    = {"pathname":..,"allowedContentTypes":..,"maximumSizeInBytes":..,"addRandomSuffix":true,"validUntil":..}
hmacHex    = HEX( HMAC-SHA256( key = READ/WRITE トークン全体, msg = base64(payload) ) )
```

実装は `server/internal/storage/blob.go` の `GenerateBlobClientToken`。

### 4.5 フロント実装

| ファイル | 役割 |
| --- | --- |
| `front/src/hooks/useVideoUpload.ts` | 本番は Blob 直接アップロード、ローカルは multipart。進捗付き |
| `front/src/api/endpoints.ts` | `uploads.token` / `uploads.blob` / `API_DIRECT_BASE_URL` |

依存: `@vercel/blob`（`upload()` を使用。大容量は `multipart: true` で自動分割）。

分岐の考え方:

- `import.meta.env.PROD` が真: ①〜④ の直接アップロード方式。
- それ以外（ローカル開発）: 従来どおり `multipart/form-data` を API に直接送る。

---

## 5. 環境変数一覧

### 5.1 API（`video-sharing-api`）

| 変数 | 必須 | 説明 |
| --- | --- | --- |
| `DATABASE_URL` | ✅ | PostgreSQL 接続文字列（本番は Neon、`sslmode=require`） |
| `BLOB_READ_WRITE_TOKEN` | 本番✅ | Vercel Blob の読み書きトークン。**設定されると保存先が Blob に切り替わる** |
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

> Blob クライアントトークンは API 側で発行するため、**フロントに `BLOB_READ_WRITE_TOKEN` は不要**。

---

## 6. デプロイ手順

### 6.0 モノレポの Root Directory（Git 連携で必須）

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

Vercel CLI は最新版を使う（コンテナビルドに新しい API バージョンが必要）。

```bash
# API（Go コンテナ）
cd server
npx vercel@latest --prod

# フロント（静的 + rewrite）
cd ../front
npx vercel@latest --prod
```

### 動作確認（本番）

1. `https://video-sharing-front.vercel.app` をスーパーリロード（キャッシュ削除）
2. ログイン → 動画アップロード
3. ブラウザ Network タブで想定の流れ:
   - `POST /api/v1/uploads/token` → `200`
   - `POST /api/v1/uploads/blob` → `200`
   - `PUT https://…blob.vercel-storage.com/…` → `200/201`（本体・直接）
   - `POST /api/v1/videos` → `201`
   - 再生時 `GET .../stream` → `307`（Blob へリダイレクト）

---

## 7. 既知の注意点

- **Blob 導入前にアップロードした動画**は DB に残るがファイル実体が無い（`/tmp` 消失）。再アップロードするか削除する。
- Vercel 本番の Gin ログに `debug mode` 警告が出る（動作には影響しない）。必要なら release モード化。
- ローカル開発は Blob を使わない。`BLOB_READ_WRITE_TOKEN` をローカルに設定すると本番 Blob を汚すため、通常は設定しない。
- **PR の Vercel チェックが Error（Build 14ms で終了）** → Root Directory が `.` のままになっていないか確認（上記 6.0）。

---

## 8. このドキュメントと他資料の関係

| 資料 | 関係 |
| --- | --- |
| [`api-design.md`](./api-design.md) | エンドポイント仕様の正本。本書はアップロード系の追加分を補足 |
| [`database-design.md`](./database-design.md) | `storage_key` は相対キーまたは Blob URL のいずれか |
| [`BACKEND_API_GUIDE.md`](./BACKEND_API_GUIDE.md) | フロントからの呼び出し手順。本番の直接アップロードも記載 |

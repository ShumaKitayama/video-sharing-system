# フロントエンド向け バックエンド API ガイド

このドキュメントは、React フロントエンド（`front/`）から Go バックエンド（`server/`）を呼び出すための実務ガイドです。  
設計の詳細は [`back/api-design.md`](../back/api-design.md) を参照してください。ここでは**起動方法・呼び出し方・フロント側の扱い方**に焦点を当てます。

---

## 目次

1. [全体像](#1-全体像)
2. [開発環境の起動](#2-開発環境の起動)
3. [フロントから API を呼ぶ基本ルール](#3-フロントから-api-を呼ぶ基本ルール)
4. [共通レスポンスとエラー処理](#4-共通レスポンスとエラー処理)
5. [認証（Cookie セッション）](#5-認証cookie-セッション)
6. [API 一覧と呼び出し例](#6-api-一覧と呼び出し例)
7. [動画アップロード](#7-動画アップロード)
8. [動画再生（ストリーミング）](#8-動画再生ストリーミング)
9. [フロントエンドのコード構成との対応](#9-フロントエンドのコード構成との対応)
10. [テスト用アカウント](#10-テスト用アカウント)
11. [よくあるトラブル](#11-よくあるトラブル)

---

## 1. 全体像

### ローカル開発

```txt
ブラウザ (React :5173)
    │  HTTP (JSON / multipart / 動画バイナリ)
    │  Cookie: session_id（ログイン時に自動付与）
    ▼
Go API サーバー (:8080)
    ├── /health, /ready          … 稼働確認（認証不要）
    └── /api/v1/*                … 業務 API
            │
            ├── PostgreSQL       … メタデータ（ユーザー・動画情報など）
            └── ローカルディスク   … 動画ファイル実体（UPLOAD_DIR）
```

### 本番（Vercel）

```txt
ブラウザ
    │  同一オリジン（フロントの rewrite で /api/* を API に中継）
    ├─ 小さな JSON / 認証 ─▶ フロント(静的) ─▶ Go API(コンテナ) ─▶ Neon PostgreSQL + Vercel Blob
    ├─ 動画本体 ──────────────────────────────────────▶ Vercel Blob（直接・API 非経由）
    └─ 再生 GET .../stream ─▶ API が 307 ─▶ Vercel Blob 公開 URL
```

本番構成・大容量アップロードの仕組みは [`back/deployment-and-storage.md`](./deployment-and-storage.md) が正本。

### バックエンドが提供する機能

| 分類 | 内容 |
| --- | --- |
| 稼働監視 | `/health`（プロセス生存）、`/ready`（DB・ストレージ接続） |
| 認証 | ログイン / ログアウト / 自分情報取得（Cookie セッション） |
| ユーザー | 先生による作成・一覧・更新・削除（論理削除） |
| 動画 | アップロード、一覧、詳細、編集、削除、自分の動画一覧 |
| 再生 | MP4/WebM の Range 対応ストリーミング |
| コメント | 一覧、投稿、削除 |
| いいね | 追加（冪等）、解除、自分の状態確認 |

### ロールと権限の概要

| ロール | できること |
| --- | --- |
| 未ログイン | 公開動画の一覧・詳細・再生、コメント一覧の閲覧 |
| `student` | 上記に加え、自分の動画投稿・編集・削除、コメント・いいね |
| `teacher` | 生徒の機能すべて、ユーザー管理、全動画の非表示/再公開 |

---

## 2. 開発環境の起動

フロントとバックエンドは**別プロセス**で動きます。通常は次の 2 段階です。

1. **バックエンド + DB** を起動する
2. **フロントエンド** を起動する

### 方法 A: Docker Compose（教室・初回セットアップ向け）

リポジトリのルートで実行します。

```bash
# リポジトリルートへ移動
cd /path/to/video-sharing-system

# DB + API をビルドして起動（フォアグラウンド）
docker compose up --build

# バックグラウンドで起動する場合
docker compose up -d --build
```

起動後の URL:

| サービス | URL |
| --- | --- |
| API | `http://localhost:8080` |
| ヘルスチェック | `http://localhost:8080/health` |
| 準備完了確認 | `http://localhost:8080/ready` |

動作確認:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

停止:

```bash
docker compose down          # コンテナ停止（データは volume に残る）
docker compose down -v       # volume も削除（DB・動画ファイルも消える）
```

#### Docker 時の注意

- 動画ファイルはコンテナ内 `/app/uploads` に保存され、Docker volume `videoshare_uploads` に永続化されます。
- CORS は `http://localhost:5173` と `http://127.0.0.1:5173` を許可する設定です（`docker-compose.yml` の `CORS_ORIGINS`）。
- フロントは Docker に含まれていないため、**別途 `front/` で `npm run dev` が必要**です。

---

### 方法 B: ネイティブ起動（フロント開発の日常向け）

#### 前提ソフトウェア

| ツール | 用途 | 推奨バージョン |
| --- | --- | --- |
| Node.js | フロントエンド | `20.19` 以上 |
| Go | バックエンド | `1.22` 以上 |
| PostgreSQL | DB | `16` 相当（Docker の db だけ使っても可） |

#### ステップ 1: データベースを用意する

**PostgreSQL を Docker だけ使う場合（API はローカル Go）:**

```bash
cd /path/to/video-sharing-system
docker compose up -d db
```

接続情報（ローカル Go から）:

```txt
postgres://postgres:postgres@localhost:5432/videoshare?sslmode=disable
```

#### ステップ 2: バックエンド API を起動する

```bash
cd server

# 環境変数（省略時は下記デフォルト）
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/videoshare?sslmode=disable"
export UPLOAD_DIR="./uploads"
export MIGRATIONS_DIR="./migrations"
export CORS_ORIGINS="http://localhost:5173,http://127.0.0.1:5173"

go run ./cmd/api
```

起動ログに `API listening on http://localhost:8080` と出れば成功です。  
マイグレーションは起動時に自動適用されます。

ローカル実行時、動画実体は `server/uploads/videos/年/月/` 配下に保存されます。

#### ステップ 3: フロントエンドを起動する

```bash
cd front

# Node バージョン（nvm 利用時）
nvm use

npm install
npm run dev
```

ブラウザで `http://localhost:5173` を開きます。

#### ステップ 4: API の接続先を設定する（任意）

デフォルトでは `http://localhost:8080` に接続します（`front/src/api/endpoints.ts`）。

別ホスト（教室 LAN の先生 PC など）に繋ぐ場合は `front/.env.local` を作成します。

```bash
# front/.env.local
VITE_API_BASE_URL=http://192.168.1.10:8080
```

変更後は Vite 開発サーバーを再起動してください。

---

### 起動パターン早見表

| やりたいこと | コマンド |
| --- | --- |
| 全部 Docker（API+DB） | ルートで `docker compose up --build` + `front` で `npm run dev` |
| DB だけ Docker、API は Go 直実行 | `docker compose up -d db` + `server` で `go run ./cmd/api` + `front` で `npm run dev` |
| 統合テスト実行 | ルートで `docker compose -f docker-compose.yml -f docker-compose.test.yml run --rm tester` |
| E2E スモーク | `docker compose up -d --build` の後 `bash scripts/e2e_smoke.sh` |

---

## 3. フロントから API を呼ぶ基本ルール

### ベース URL

```txt
{API_BASE_URL}/api/v1/...
```

- 開発時の既定値: `http://localhost:8080`
- フロントの定数: [`front/src/api/endpoints.ts`](../front/src/api/endpoints.ts) の `API_BASE_URL`

### 認証方式

- **JWT ではなく HttpOnly Cookie**（Cookie 名: `session_id`）
- ログイン成功時にサーバーが Set-Cookie する
- 以降のリクエストでブラウザが自動送信する

そのため、**`fetch` には必ず `credentials: "include"` を付ける**必要があります。

```typescript
const res = await fetch(`${API_BASE_URL}/api/v1/auth/me`, {
  credentials: "include",
});
```

`axios` を使う場合は `withCredentials: true` です。

### CORS

バックエンドは `CORS_ORIGINS` に登録されたオリジンからのみ、Cookie 付きリクエストを受け付けます。  
フロントのオリジン（例: `http://localhost:5173`）がリストに無いとログイン Cookie が使えません。

### Content-Type

| 用途 | Content-Type |
| --- | --- |
| JSON API | `application/json` |
| 動画アップロード | `multipart/form-data`（ブラウザが自動設定） |
| 動画再生 | サーバーが `video/mp4` / `video/webm` を返す |

### ID と日時

- 公開 ID は **UUID 文字列**（内部の数値 ID は API に出ない）
- 日時は **RFC3339**（例: `2026-05-18T10:00:00+09:00`）

### リクエスト ID

すべてのレスポンスに `X-Request-ID` ヘッダーが付きます。エラー調査時にサーバーログと突き合わせてください。

---

## 4. 共通レスポンスとエラー処理

### 単体成功

```json
{
  "data": { ... }
}
```

### 一覧成功

```json
{
  "data": [ ... ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

### エラー

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "入力内容を確認してください",
    "details": [
      { "field": "title", "message": "1〜80文字で入力してください" }
    ],
    "request_id": "req_01"
  }
}
```

### 主なエラーコード

| HTTP | code | フロントでの扱い例 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | フォームの該当フィールドに `details[].message` を表示 |
| 401 | `UNAUTHENTICATED` | ログインモーダルを開く / ログイン画面へ誘導 |
| 403 | `FORBIDDEN` | 「権限がありません」と表示 |
| 404 | `NOT_FOUND` | 「見つかりません」、一覧へ戻す |
| 409 | `CONFLICT` | 「すでに存在します」など |
| 413 | `PAYLOAD_TOO_LARGE` | 「ファイルが大きすぎます」（上限 500MB） |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | 「MP4 または WebM のみ」 |
| 429 | `RATE_LIMITED` | 「しばらく待ってから再試行」 |
| 500 | `INTERNAL_ERROR` | 汎用エラーメッセージ |
| 503 | `SERVICE_UNAVAILABLE` | 「サーバー準備中」 |

### 推奨: 共通 fetch ラッパー（実装イメージ）

`front/src/services/api.ts` に置く想定のパターンです（学生は直接 `fetch` しない）。

```typescript
import { API_BASE_URL } from "../api/endpoints";

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    public details?: { field: string; message: string }[],
  ) {
    super(code);
  }
}

export async function apiFetch<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...init.headers,
    },
  });

  // 204 No Content
  if (res.status === 204) {
    return undefined as T;
  }

  const body = await res.json();

  if (!res.ok) {
    throw new ApiError(
      res.status,
      body.error?.code ?? "INTERNAL_ERROR",
      body.error?.details,
    );
  }

  return body.data as T;
}
```

---

## 5. 認証（Cookie セッション）

### ログイン

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "student01",
  "password": "classroom-pass"
}
```

**成功 `200`**

```json
{
  "data": {
    "user": {
      "id": "9897403e-fd1e-45fe-b1ae-2a67c0a9d02e",
      "username": "student01",
      "display_name": "山田 太郎",
      "role": "student"
    }
  }
}
```

レスポンスと同時に `session_id` Cookie が Set されます（HttpOnly, SameSite=Lax, 有効期限 24 時間）。

**失敗**

- `401 UNAUTHENTICATED` … ユーザー名またはパスワード不一致
- `403 FORBIDDEN` … アカウント無効化済み

### 自分情報取得

```http
GET /api/v1/auth/me
```

ログイン必須。`200` でユーザー JSON、`401` で未ログイン。

### ログアウト

```http
POST /api/v1/auth/logout
```

成功時 `204`（本文なし）。Cookie がクリアされます。

### フロント実装のポイント

- [`front/src/hooks/useAuth.ts`](../front/src/hooks/useAuth.ts) は現在モックです。接続時は上記 3 エンドポイントを呼ぶよう差し替えます。
- ページ読み込み時に `GET /auth/me` でセッション復元するのが一般的です。
- **生徒向け UI コンポーネント内で `fetch` を書かない**（AGENTS.md のルール）。必ず hook / service 経由にします。

---

## 6. API 一覧と呼び出し例

以下、パスはすべて `/api/v1` 以降です。`🔒` はログイン必須、`👨‍🏫` は先生のみ。

### 稼働確認（認証不要）

| メソッド | パス | 説明 |
| --- | --- | --- |
| GET | `/health` | プロセス生存確認 |
| GET | `/ready` | DB・ストレージの準備確認 |

※ ベース URL の直下（`/api/v1` の外）です。

```typescript
// 例: 起動確認
const ok = await fetch(`${API_BASE_URL}/ready`).then((r) => r.ok);
```

---

### ユーザー

| メソッド | パス | 権限 | 説明 |
| --- | --- | --- | --- |
| GET | `/users` | 👨‍🏫 | ユーザー一覧（ページング・検索） |
| POST | `/users` | 👨‍🏫 | ユーザー作成 |
| GET | `/users/:id` | 🔒 本人 or 👨‍🏫 | ユーザー詳細 |
| PATCH | `/users/:id` | 🔒 本人 or 👨‍🏫 | ユーザー更新 |
| DELETE | `/users/:id` | 👨‍🏫 | 論理削除 |

#### ユーザー作成（先生）

```http
POST /api/v1/users
Content-Type: application/json

{
  "username": "student01",
  "display_name": "山田 太郎",
  "password": "classroom-pass",
  "role": "student"
}
```

成功 `201`:

```json
{
  "data": {
    "id": "...",
    "username": "student01",
    "display_name": "山田 太郎",
    "role": "student",
    "is_active": true
  }
}
```

#### ユーザー一覧クエリ

| パラメータ | 説明 |
| --- | --- |
| `page` | ページ番号（既定 `1`） |
| `per_page` | 1 ページ件数（既定 `20`、最大 `50`） |
| `role` | `student` / `teacher` |
| `is_active` | `true` / `false` |
| `q` | username / display_name の部分一致 |

---

### 動画

| メソッド | パス | 権限 | 説明 |
| --- | --- | --- | --- |
| GET | `/videos` | 公開 | 動画一覧 |
| POST | `/videos` | 🔒 | 動画アップロード |
| GET | `/videos/:id` | 公開* | 動画詳細 |
| PATCH | `/videos/:id` | 🔒 投稿者 or 👨‍🏫 | メタデータ更新 |
| DELETE | `/videos/:id` | 🔒 投稿者 or 👨‍🏫 | 削除（論理削除＋ファイル削除） |
| GET | `/me/videos` | 🔒 | 自分の動画一覧（非表示含む） |
| GET | `/videos/:id/stream` | 公開* | 動画本体配信 |

\* 非表示（`hidden`）動画は投稿者本人と先生のみ参照・再生可能。それ以外は `404`。

#### 動画一覧クエリ

| パラメータ | 説明 |
| --- | --- |
| `page` | ページ番号（既定 `1`） |
| `per_page` | 1 ページ件数（既定 `20`、最大 `50`） |
| `q` | タイトル・説明・投稿者名の部分一致 |
| `sort` | `latest`（既定）/ `oldest` / `most_viewed` / `most_liked` |
| `uploader_id` | 投稿者 UUID で絞り込み |
| `status` | 👨‍🏫 のみ: `published` / `hidden` / `all` |

#### 動画オブジェクトの形（詳細・一覧共通）

[`front/src/types/index.ts`](../front/src/types/index.ts) の `Video` 型と一致します。

```json
{
  "id": "1f1d3fb8-5b54-48b0-83c5-d5f5c52d82e9",
  "title": "理科の実験",
  "description": "水の状態変化を撮影した動画",
  "status": "published",
  "uploader": {
    "id": "9897403e-fd1e-45fe-b1ae-2a67c0a9d02e",
    "display_name": "山田 太郎"
  },
  "mime_type": "video/mp4",
  "file_size_bytes": 1024000,
  "duration_seconds": null,
  "view_count": 12,
  "like_count": 3,
  "comment_count": 2,
  "created_at": "2026-05-18T10:00:00+09:00",
  "updated_at": "2026-05-18T10:05:00+09:00"
}
```

#### 動画更新

投稿者は `title` / `description` のみ。先生は加えて `status`（`published` / `hidden`）も変更可能。

```http
PATCH /api/v1/videos/:id
Content-Type: application/json

{
  "title": "理科の実験 その1",
  "description": "更新後の説明"
}
```

---

### コメント

| メソッド | パス | 権限 | 説明 |
| --- | --- | --- | --- |
| GET | `/videos/:id/comments` | 公開* | コメント一覧 |
| POST | `/videos/:id/comments` | 🔒 | コメント投稿 |
| DELETE | `/comments/:id` | 🔒 本人 or 👨‍🏫 | コメント削除 |

#### コメント投稿

```http
POST /api/v1/videos/:videoId/comments
Content-Type: application/json

{ "body": "分かりやすかった" }
```

成功 `201`。`body` は 1〜300 文字。

#### コメントオブジェクト

[`front/src/types/index.ts`](../front/src/types/index.ts) の `Comment` 型と一致します。

```json
{
  "id": "d369ef95-d7da-4ef2-8b44-4452f1da8514",
  "body": "分かりやすかった",
  "author": {
    "id": "9897403e-fd1e-45fe-b1ae-2a67c0a9d02e",
    "display_name": "山田 太郎"
  },
  "created_at": "2026-05-18T10:30:00+09:00",
  "updated_at": "2026-05-18T10:30:00+09:00"
}
```

---

### いいね

| メソッド | パス | 権限 | 説明 |
| --- | --- | --- | --- |
| GET | `/videos/:id/like` | 🔒 | 自分がいいね済みか |
| PUT | `/videos/:id/like` | 🔒 | いいね追加（冪等） |
| DELETE | `/videos/:id/like` | 🔒 | いいね解除（冪等） |

#### いいね追加

```http
PUT /api/v1/videos/:id/like
```

成功 `200`:

```json
{
  "data": {
    "liked": true,
    "like_count": 4
  }
}
```

[`front/src/hooks/useLike.ts`](../front/src/hooks/useLike.ts) は現在 `toggleLike` 1 本ですが、バックエンドは **PUT / DELETE の 2 本**です。接続時は liked 状態に応じて呼び分けてください。

---

## 7. 動画アップロード

### エンドポイント

```http
POST /api/v1/videos
Content-Type: multipart/form-data
```

### フォームフィールド

| フィールド | 必須 | 説明 |
| --- | --- | --- |
| `file` | はい | MP4 または WebM |
| `title` | はい | 1〜80 文字 |
| `description` | いいえ | 0〜1000 文字 |

### 制約

- MIME: `video/mp4`, `video/webm`
- 拡張子: `.mp4`, `.webm`（MIME と一致必須）
- 最大サイズ: **500MB**
- レート制限: ユーザーあたり 1 分 5 回

### 成功 `201`

```json
{
  "data": {
    "id": "1f1d3fb8-5b54-48b0-83c5-d5f5c52d82e9",
    "title": "理科の実験",
    "status": "published"
  }
}
```

### フロント実装例

[`front/src/hooks/useVideoUpload.ts`](../front/src/hooks/useVideoUpload.ts) から呼ぶ想定です。

```typescript
import { API_BASE_URL } from "../api/endpoints";
import { endpoints } from "../api/endpoints";

async function uploadVideo(file: File, title: string, description: string) {
  const form = new FormData();
  form.append("file", file);
  form.append("title", title.trim());
  form.append("description", description);

  const res = await fetch(`${API_BASE_URL}${endpoints.videos.create}`, {
    method: "POST",
    credentials: "include",
    body: form,
    // Content-Type は付けない（ブラウザが boundary 付きで自動設定）
  });

  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error?.message ?? "アップロードに失敗しました");
  }

  const { data } = await res.json();
  return data; // { id, title, status }
}
```

### 本番（Vercel）での大容量アップロード

Vercel には**リクエストボディ約 4.5MB の上限**があり、動画本体を `POST /videos` に送ると `413` になる（ブラウザ上は CORS エラーに見える）。そのため本番では、ブラウザから **Cloudflare R2 へ動画本体を直接アップロード**し、完了後に小さな JSON だけを API に送る。

```txt
① POST /api/v1/uploads/token   … 短命トークン取得（Cookie）
② POST /api/v1/uploads/direct  … 署名付きアップロードURL取得 { content_type }
③ PUT → Cloudflare R2          … 動画本体を直接アップロード（API 非経由・進捗取得可）
④ POST /api/v1/videos (JSON)   … { blob_url, title, description, content_type, duration_seconds }
```

④ の `blob_url` には ② で返ってきた `public_url` を入れる（フィールド名は旧方式との互換で据え置き）。

この分岐は [`front/src/hooks/useVideoUpload.ts`](../front/src/hooks/useVideoUpload.ts) に実装済みで、`import.meta.env.PROD` で自動的に切り替わる（ローカルは上記 multipart 方式）。**学生・UI 側は `uploadVideo()` を呼ぶだけ**でよい。詳細は [`back/deployment-and-storage.md`](./deployment-and-storage.md) 第4章。

外部依存は無い（`XMLHttpRequest` のみ）。

### アップロード進捗について

- 直接アップロードも multipart も `XMLHttpRequest` の `upload.onprogress` で進捗を取得する。
- いずれも `useVideoUpload` の `progress`（0〜100）で受け取れる。

---

## 8. 動画再生（ストリーミング）

### 配信 URL

```txt
GET {API_BASE_URL}/api/v1/videos/{id}/stream
```

### フロントでの使い方（`<video>` 要素）

ブラウザの `<video>` は **Range Request を自動で送る**ため、特別な JS は不要です。`src` に URL を指定するだけで部分配信（206）が動きます。

```tsx
import { API_BASE_URL } from "../api/endpoints";
import { endpoints } from "../api/endpoints";

function VideoPlayer({ videoId }: { videoId: string }) {
  const src = `${API_BASE_URL}${endpoints.videos.stream(videoId)}`;

  return (
    <video
      controls
      src={src}
      // Cookie 付きで読み込むため crossOrigin は通常不要
      // LAN 上の別オリジンで CORS + Cookie が絡む場合は要検証
    />
  );
}
```

[`front/src/components/internal/VideoPlayer.tsx`](../front/src/components/internal/VideoPlayer.tsx) は現在プレースホルダーです。接続時に上記のように `src` を設定してください。

### 挙動

| 状況 | HTTP ステータス |
| --- | --- |
| 全体取得 | `200 OK` |
| 有効な Range | `206 Partial Content` |
| 不正な Range | `416 Range Not Satisfiable` |

レスポンスヘッダー例（Range 時）:

```txt
Accept-Ranges: bytes
Content-Range: bytes 0-1048575/73400320
Content-Length: 1048576
Content-Type: video/mp4
```

再生開始時に `view_count` が 1 増えます（同一視聴者の短時間連打は除外）。

### 本番（Vercel）の挙動

本番では動画実体が Vercel Blob にあるため、`GET .../stream` は **`307` で Blob の公開 URL にリダイレクト**する。`<video src>` はブラウザが自動的にリダイレクト先を辿るので、フロント側の実装変更は不要。

| 保存先 | ステータス |
| --- | --- |
| ローカルディスク | `200` / `206`（Range 配信） |
| Vercel Blob | `307`（Blob 公開 URL へ） |

### 注意

- 動画 URL を `fetch` で丸ごと読み込まないこと（メモリ枯渇の原因）
- `<video src="...">` または `<source src="...">` でブラウザに任せる
- 非表示動画は投稿者・先生以外 `404` になる

---

## 9. フロントエンドのコード構成との対応

AGENTS.md / プロジェクトルールに従い、層を分けて実装します。

```txt
front/src/
├── api/endpoints.ts      … URL 定義（既存）
├── services/api.ts       … fetch 実装（現在モック → ここを差し替え）
├── hooks/                … UI が使う唯一の API 入口
│   ├── useAuth.ts
│   ├── useVideos.ts
│   ├── useVideoDetail.ts
│   ├── useVideoUpload.ts
│   ├── useComments.ts
│   └── useLike.ts
├── types/index.ts        … レスポンス型（バックエンドと整合済み）
└── components/
    ├── student/          … 学生が編集してよい UI
    └── internal/         … プレーヤー・レイアウトなど保護領域
```

### 接続作業のチェックリスト

| 順番 | 作業 | ファイル |
| --- | --- | --- |
| 1 | 共通 `apiFetch` とエラークラスを実装 | `services/api.ts` |
| 2 | 認証を実 API に接続 | `hooks/useAuth.ts` |
| 3 | 動画一覧・詳細 | `services/api.ts` → `useVideos.ts`, `useVideoDetail.ts` |
| 4 | アップロード | `hooks/useVideoUpload.ts` |
| 5 | コメント取得・投稿 | `services/api.ts`, `useComments.ts` |
| 6 | いいね PUT/DELETE | `services/api.ts`, `useLike.ts` |
| 7 | プレーヤーに stream URL を設定 | `components/internal/VideoPlayer.tsx` |

### 型の対応

`front/src/types/index.ts` はバックエンドの JSON 形に合わせて定義済みです。接続時に大きな型変更は不要な想定です。

---

## 10. テスト用アカウント

### マイグレーションで自動作成されるユーザー

初回起動時、DB に `teacher` ユーザー（表示名「先生」）が 1 件だけ作成されます。  
パスワードの平文はリポジトリに記載されていないため、**そのままではログインできません**。

### E2E / 開発用テストユーザー（推奨）

以下の SQL スクリプトで投入できます（パスワードは両方とも同じ）。

```bash
docker compose exec -T db psql -U postgres -d videoshare \
  < scripts/seed_test_users.sql
```

| username | password | role |
| --- | --- | --- |
| `teacher_test` | `e2e-pass-123` | teacher |
| `student_test` | `e2e-pass-123` | student |

### 新規ユーザーの作り方（本番的な流れ）

1. `teacher_test` でログイン
2. `POST /api/v1/users` で生徒アカウントを作成
3. 作成した生徒でログインして動画投稿

設計書の例ではパスワード `classroom-pass` を使っていますが、実際の値は自由です（8〜72 文字）。

---

## 11. よくあるトラブル

### ログインしてもすぐ未ログインになる

- `fetch` に `credentials: "include"` が付いているか確認
- フロントのオリジンが `CORS_ORIGINS` に含まれているか確認
- `localhost` と `127.0.0.1` は別オリジン扱いです。どちらかに統一してください

### CORS エラーが出る

`server` 起動時の環境変数 `CORS_ORIGINS` に、ブラウザのアドレスバーのオリジンを追加します。

```bash
export CORS_ORIGINS="http://localhost:5173,http://127.0.0.1:5173,http://192.168.1.10:5173"
```

Docker の場合は `docker-compose.yml` の `api.environment.CORS_ORIGINS` を編集して再起動します。

### `ready` が 503 になる

- PostgreSQL が起動しているか（`docker compose ps`）
- `DATABASE_URL` が正しいか
- `uploads` ディレクトリに書き込み権限があるか

### 動画が再生されない

- アップロードが成功しているか（`POST /videos` が 201 か）
- `<video src>` の URL が `.../videos/{id}/stream` になっているか
- 非表示動画を他人が再生しようとしていないか（404）

### アップロードが 415 になる

- ファイルが本当に MP4 / WebM か（拡張子と MIME の両方が一致必要）
- 空ファイルや壊れたファイルは拒否されます

### フロントだけ動いてモックデータが出る

現在 [`front/src/services/api.ts`](../front/src/services/api.ts) は**モック実装**です。バックエンド接続後、このファイルを実 API 呼び出しに差し替える必要があります。UI（`components/student/`）は hooks 経由なので、多くの場合 UI 側の変更は不要です。

---

## 参考リンク

| 資料 | 内容 |
| --- | --- |
| [`back/api-design.md`](../back/api-design.md) | API 設計の正本 |
| [`back/database-design.md`](../back/database-design.md) | DB スキーマ |
| [`back/deployment-and-storage.md`](../back/deployment-and-storage.md) | Vercel 本番構成・Blob・大容量アップロード |
| [`AGENTS.md`](../AGENTS.md) | 編集可能領域・保護領域のルール |
| [`front/src/api/endpoints.ts`](../front/src/api/endpoints.ts) | エンドポイント定数 |
| [`scripts/e2e_smoke.sh`](../scripts/e2e_smoke.sh) | API 動作の自動スモークテスト |

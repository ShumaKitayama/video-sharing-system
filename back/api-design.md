# API設計書

## 1. 基本方針

### 1.1 APIの役割

APIは、フロントエンドが安全に使える「決まった窓口」を提供する。  
学生が編集するUI側からは、生の `fetch` やmultipart処理を直接扱わせず、将来 `frontend/src/hooks/` や `frontend/src/services/` からこのAPIを呼ぶ前提とする。

### 1.2 ベースURL

```txt
/api/v1
```

### 1.3 通信形式

| 用途 | Content-Type |
| --- | --- |
| 通常JSON | `application/json` |
| 動画アップロード（ローカル） | `multipart/form-data` |
| 動画アップロード（本番 Blob 登録） | `application/json` |
| 動画配信 | 動画のMIME型に応じて `video/mp4` / `video/webm`（本番は Blob へ 307 リダイレクト） |

> 本番（Vercel）では、動画本体は API を経由せずブラウザから Vercel Blob へ直接アップロードする。詳細は [`deployment-and-storage.md`](./deployment-and-storage.md) を参照。

### 1.4 IDと日時

| 項目 | 形式 |
| --- | --- |
| 公開ID | UUID文字列 |
| 日時 | RFC3339 |
| 内部DB ID | APIには出さない |

---

## 2. 認証と権限

### 2.1 認証方式

初期版は、HTTP Only Cookieを使う単純なセッション方式とする。

- Cookie名: `session_id`
- `HttpOnly`: `true`
- `SameSite`: `Lax`
- `Secure`: 本番相当のHTTPS環境だけ `true`
- セッション有効期限: 24時間

### 2.2 採用しないもの

次は初期版では使わない。

- JWT
- refresh token
- OAuth
- 外部IdP

理由は、LAN内授業用途では構造が重く、教育効果より実装複雑性の方が先に増えるためである。

### 2.3 ロール

| ロール | できること |
| --- | --- |
| `student` | 自分の動画投稿、編集、削除、コメント、いいね |
| `teacher` | 生徒機能すべて、ユーザー管理、全動画の非表示/再公開 |

### 2.4 権限表

| 操作 | 未ログイン | student | teacher |
| --- | --- | --- | --- |
| 公開動画一覧 | 可 | 可 | 可 |
| 公開動画詳細 | 可 | 可 | 可 |
| 動画再生 | 可 | 可 | 可 |
| ログイン | 可 | 可 | 可 |
| 自分情報取得 | 不可 | 可 | 可 |
| 動画投稿 | 不可 | 可 | 可 |
| 自分の動画編集 | 不可 | 可 | 可 |
| 他人の動画編集 | 不可 | 不可 | 可 |
| コメント投稿 | 不可 | 可 | 可 |
| コメント削除 | 不可 | 自分のみ | 全件 |
| ユーザー管理 | 不可 | 不可 | 可 |
| 動画非表示 | 不可 | 不可 | 可 |

---

## 3. 共通レスポンス

### 3.1 単体成功

```json
{
  "data": {
    "id": "7bd2dcb4-63f5-4f86-8c5c-9600975d6f1f"
  }
}
```

### 3.2 一覧成功

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 0,
    "total_pages": 0
  }
}
```

### 3.3 エラー

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "入力内容を確認してください",
    "details": [
      {
        "field": "title",
        "message": "1文字以上80文字以下で入力してください"
      }
    ],
    "request_id": "req_01"
  }
}
```

### 3.4 共通エラーコード

| HTTP | code | 用途 |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | 入力不正 |
| 401 | `UNAUTHENTICATED` | 未ログイン |
| 403 | `FORBIDDEN` | 権限不足 |
| 404 | `NOT_FOUND` | 対象なし |
| 409 | `CONFLICT` | 重複、状態衝突 |
| 413 | `PAYLOAD_TOO_LARGE` | 動画サイズ超過（※本番 Vercel ではプラットフォーム層がボディ約4.5MBで返すことがある。回避策は [`deployment-and-storage.md`](./deployment-and-storage.md) 第2.3章） |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | 非対応動画形式 |
| 429 | `RATE_LIMITED` | 短時間連打 |
| 500 | `INTERNAL_ERROR` | サーバー内部異常 |
| 503 | `SERVICE_UNAVAILABLE` | DB未接続など |

---

## 4. 共通仕様

### 4.1 ページング

| パラメータ | 既定値 | 制約 |
| --- | --- | --- |
| `page` | `1` | 1以上 |
| `per_page` | `20` | 1〜50 |

### 4.2 並び順

動画一覧の既定順は `created_at desc` とする。

利用可能な値:

- `latest`
- `oldest`
- `most_viewed`
- `most_liked`

### 4.3 検索

`q` が指定された場合、以下を部分一致検索対象にする。

- 動画タイトル
- 動画説明
- 投稿者表示名

### 4.4 バリデーション

| 項目 | 制約 |
| --- | --- |
| username | 3〜32文字、英数字と `_` のみ |
| display_name | 1〜40文字 |
| password | 8〜72文字 |
| title | 1〜80文字 |
| description | 0〜1000文字 |
| comment body | 1〜300文字 |
| video file size | 500MB以下 |
| video MIME | `video/mp4`, `video/webm` |

### 4.5 リクエストID

すべてのレスポンスに `X-Request-ID` を付ける。  
クライアントが付けてきた値が無ければサーバー側で生成し、ログとエラーレスポンスにも同じ値を載せる。

### 4.6 軽量レート制限

教室LAN内でも連打による事故を避けるため、初期版では次だけ軽く制限する。

| 対象 | 目安 |
| --- | --- |
| ログイン | IPごとに1分10回 |
| 動画アップロード | ユーザーごとに1分5回 |
| コメント投稿 | ユーザーごとに1分30回 |

単一サーバー前提なので、初期版はメモリ内カウンタでよい。分散レート制限は採用しない。

---

## 5. 稼働確認API

### 5.1 `GET /health`

プロセスが起動しているかだけを返す。

#### Response `200`

```json
{
  "data": {
    "status": "ok"
  }
}
```

### 5.2 `GET /ready`

DB接続、保存ディレクトリの書き込み可否など、実処理に必要な依存先を確認する。

#### Response `200`

```json
{
  "data": {
    "status": "ready",
    "database": "ok",
    "storage": "ok"
  }
}
```

#### Response `503`

```json
{
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "依存サービスの準備ができていません",
    "details": [
      {
        "field": "database",
        "message": "connection failed"
      }
    ],
    "request_id": "req_01"
  }
}
```

---

## 6. 認証API

### 6.1 `POST /auth/login`

#### Request

```json
{
  "username": "student01",
  "password": "classroom-pass"
}
```

#### Response `200`

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

#### 主な失敗

- `401 UNAUTHENTICATED`: ユーザー名またはパスワード不一致
- `403 FORBIDDEN`: 無効化済みユーザー

### 6.2 `POST /auth/logout`

現在のセッションを削除する。

#### Response `204`

本文なし。

### 6.3 `GET /auth/me`

#### Response `200`

```json
{
  "data": {
    "id": "9897403e-fd1e-45fe-b1ae-2a67c0a9d02e",
    "username": "student01",
    "display_name": "山田 太郎",
    "role": "student"
  }
}
```

---

## 7. ユーザーAPI

### 7.1 ユーザー表現

```json
{
  "id": "9897403e-fd1e-45fe-b1ae-2a67c0a9d02e",
  "username": "student01",
  "display_name": "山田 太郎",
  "role": "student",
  "is_active": true,
  "created_at": "2026-05-18T10:00:00+09:00",
  "updated_at": "2026-05-18T10:00:00+09:00"
}
```

### 7.2 `GET /users`

先生のみ。ユーザー一覧を返す。

#### Query

| 名前 | 説明 |
| --- | --- |
| `page` | ページ番号 |
| `per_page` | 1ページ件数 |
| `role` | `student` / `teacher` |
| `is_active` | `true` / `false` |
| `q` | username または display_name |

### 7.3 `POST /users`

先生のみ。新規ユーザーを作る。

#### Request

```json
{
  "username": "student01",
  "display_name": "山田 太郎",
  "password": "classroom-pass",
  "role": "student"
}
```

#### Response `201`

```json
{
  "data": {
    "id": "9897403e-fd1e-45fe-b1ae-2a67c0a9d02e",
    "username": "student01",
    "display_name": "山田 太郎",
    "role": "student",
    "is_active": true
  }
}
```

### 7.4 `GET /users/:id`

先生、または本人のみ。

### 7.5 `PATCH /users/:id`

本人は `display_name` と `password` を更新できる。  
先生は `display_name`, `password`, `role`, `is_active` を更新できる。

#### Request例

```json
{
  "display_name": "山田 たろう"
}
```

### 7.6 `DELETE /users/:id`

先生のみ。物理削除ではなく論理削除を行う。

#### Response `204`

本文なし。

---

## 8. 動画API

### 8.1 動画表現

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
  "duration_seconds": 42,
  "view_count": 12,
  "like_count": 3,
  "comment_count": 2,
  "created_at": "2026-05-18T10:00:00+09:00",
  "updated_at": "2026-05-18T10:05:00+09:00"
}
```

### 8.2 `GET /videos`

公開動画一覧を返す。

#### Query

| 名前 | 説明 |
| --- | --- |
| `page` | ページ番号 |
| `per_page` | 1ページ件数 |
| `q` | タイトル、説明、投稿者名検索 |
| `sort` | `latest`, `oldest`, `most_viewed`, `most_liked` |
| `uploader_id` | 投稿者絞り込み |
| `status` | 先生のみ `published`, `hidden`, `all` を指定可能 |

### 8.3 `POST /videos`

ログイン必須。動画を登録する。**Content-Type によって 2 つの入力形式**を受け付ける。

#### 形式 A: `multipart/form-data`（ローカル開発の従来方式）

| フィールド | 必須 | 説明 |
| --- | --- | --- |
| `file` | 必須 | MP4またはWebM |
| `title` | 必須 | 1〜80文字 |
| `description` | 任意 | 0〜1000文字 |
| `duration_seconds` | 任意 | 再生時間（秒） |

実装上の注意:

1. `handler` でファイルを丸ごとメモリに読まない
2. `storage` へ固定長バッファで逐次保存する
3. DB保存に失敗した場合は保存済みファイルを削除する
4. MIME型と拡張子は両方確認する

#### 形式 B: `application/json`（本番: Blob 直接アップロード後の登録）

ブラウザが先に Vercel Blob へ動画本体を直接アップロードし、その公開 URL だけを登録する。

```json
{
  "blob_url": "https://<store>.public.blob.vercel-storage.com/<uuid>.mp4",
  "title": "理科の実験",
  "description": "水の状態変化",
  "content_type": "video/mp4",
  "duration_seconds": 42
}
```

サーバー側の検証:

1. `blob_url` のホストが `*.blob.vercel-storage.com` であること
2. HEAD で実体を確認し、サイズ（500MB以下）と MIME（MP4/WebM）が妥当であること

事前フローの詳細は [`deployment-and-storage.md`](./deployment-and-storage.md) 第4章を参照。

#### Response `201`（両形式共通）

```json
{
  "data": {
    "id": "1f1d3fb8-5b54-48b0-83c5-d5f5c52d82e9",
    "title": "理科の実験",
    "status": "published"
  }
}
```

### 8.4 `GET /videos/:id`

公開動画詳細を返す。  
非表示動画は、投稿者本人または先生だけ参照できる。

### 8.5 `PATCH /videos/:id`

投稿者本人または先生のみ。

#### Request例

```json
{
  "title": "理科の実験 その1",
  "description": "更新後の説明"
}
```

先生のみ、次も更新できる。

```json
{
  "status": "hidden"
}
```

### 8.6 `DELETE /videos/:id`

投稿者本人または先生のみ。  
DBは論理削除、動画ファイルは削除キューなしで即時削除する単純方式とする。

#### Response `204`

本文なし。

### 8.7 `GET /me/videos`

ログイン中ユーザーが投稿した動画一覧を返す。  
自分の非表示動画も取得できる。

### 8.8 `POST /uploads/token`

ログイン必須（Cookie）。大容量アップロード用の短命トークン（HMAC 署名・15分）を発行する。  
本番でブラウザが Blob へ直接アップロードする際の認可に使う。

#### Response `200`

```json
{
  "data": {
    "token": "12:student:1783085641:9f3c..."
  }
}
```

### 8.9 `POST /uploads/blob`

`@vercel/blob` クライアントアップロードの「トークン発行窓口」（`handleUploadUrl`）。  
認可は Cookie または `8.8` のトークン（リクエストの `clientPayload`）で行う。動画本体はここを通らない。

- `type: "blob.generate-client-token"`: Blob クライアントトークンを返す

```json
{
  "type": "blob.generate-client-token",
  "clientToken": "vercel_blob_client_<storeId>_..."
}
```

> このエンドポイントは共通レスポンス（`data` ラップ）ではなく、`@vercel/blob` が要求する固定形状で返す点に注意。詳細は [`deployment-and-storage.md`](./deployment-and-storage.md) 第4章。

---

## 9. ストリーミングAPI

### 9.1 `GET /videos/:id/stream`

動画本体を返す。ブラウザの `<video>` 要素から呼ばれる想定である。

### 9.2 Range Request対応

#### Request Header例

```txt
Range: bytes=0-1048575
```

#### Response Header例

```txt
Accept-Ranges: bytes
Content-Range: bytes 0-1048575/73400320
Content-Length: 1048576
Content-Type: video/mp4
```

#### Response Status

| 状態 | HTTP |
| --- | --- |
| Rangeなし | `200 OK` |
| 有効なRangeあり | `206 Partial Content` |
| 不正Range | `416 Range Not Satisfiable` |

### 9.3 実装上の注意

- 動画を丸ごとメモリへ読み込まない
- `http.ServeContent` または同等のRange処理を使う
- 非表示動画は投稿者本人または先生のみ再生可能
- 再生開始時に再生回数を1増やす

### 9.4 保存先による分岐

`storage_key` の形で配信方法が変わる。

| `storage_key` | 挙動 |
| --- | --- |
| 相対キー（例 `videos/2026/07/<uuid>.mp4`） | ディスクから Range 配信（`200` / `206`） |
| `https://...` の Blob URL | `307 Temporary Redirect` で Blob の公開 URL へ |

本番のストリームログに `307` が出るのは正常。詳細は [`deployment-and-storage.md`](./deployment-and-storage.md) 第3章。

---

## 10. コメントAPI

### 10.1 コメント表現

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

### 10.2 `GET /videos/:id/comments`

公開コメント一覧を返す。

#### Query

| 名前 | 説明 |
| --- | --- |
| `page` | ページ番号 |
| `per_page` | 1ページ件数 |

### 10.3 `POST /videos/:id/comments`

ログイン必須。

#### Request

```json
{
  "body": "分かりやすかった"
}
```

### 10.4 `DELETE /comments/:id`

投稿者本人または先生のみ。  
論理削除を使い、件数カウンタを同一トランザクションで減らす。

---

## 11. いいねAPI

### 11.1 `PUT /videos/:id/like`

ログイン必須。  
未いいねなら追加し、すでに付いている場合は冪等に成功を返す。

#### Response `200`

```json
{
  "data": {
    "liked": true,
    "like_count": 4
  }
}
```

### 11.2 `DELETE /videos/:id/like`

ログイン必須。  
いいねが存在すれば解除し、存在しなくても成功を返す。

### 11.3 `GET /videos/:id/like`

ログイン中ユーザーがその動画をいいね済みか返す。

```json
{
  "data": {
    "liked": true
  }
}
```

---

## 12. 管理用API

### 12.1 `PATCH /videos/:id`

先生が `status` を更新して非表示/再公開を行う。

### 12.2 `GET /users`

先生がアカウント管理に使う。

### 12.3 `PATCH /users/:id`

先生が `is_active` を切り替えられる。

### 12.4 管理APIでやらないこと

- 複雑な承認フロー
- 複数段階レビュー
- 監査ログのUI化

必要性が出た段階で追加する。

---

## 13. ステータスコード整理

| 操作 | 成功 |
| --- | --- |
| 一覧取得 | `200` |
| 詳細取得 | `200` |
| 作成 | `201` |
| 更新 | `200` |
| 削除 | `204` |
| ログアウト | `204` |
| 動画配信 | `200` / `206` |

---

## 14. セキュリティ上の最低限

### 14.1 入力

- HTMLタグは保存前に必要以上に変形せず、表示側でエスケープする
- 文字数制限を必ず持つ
- ファイル拡張子とMIME型を両方検査する
- パス文字列はクライアント入力から直接作らない

### 14.2 認証

- パスワードはハッシュ化して保存する
- セッションIDはDBに平文保存せずハッシュ保存する
- 無効化ユーザーはログイン不可

### 14.3 配信

- 任意パス読み出しを許さない
- DBに登録された `storage_key` だけを使う
- ファイルが存在しない場合は `404`

---

## 15. 実装単位への分解

| handler | service | repository |
| --- | --- | --- |
| `AuthHandler` | `AuthService` | `SessionRepository`, `UserRepository` |
| `UserHandler` | `UserService` | `UserRepository` |
| `VideoHandler` | `VideoService` | `VideoRepository` |
| `StreamHandler` | `StreamService` | `VideoRepository` |
| `CommentHandler` | `CommentService` | `CommentRepository`, `VideoRepository` |
| `LikeHandler` | `LikeService` | `LikeRepository`, `VideoRepository` |

---

## 16. 最小ルーティング一覧

```txt
GET     /health
GET     /ready

POST    /api/v1/auth/login
POST    /api/v1/auth/logout
GET     /api/v1/auth/me

GET     /api/v1/users
POST    /api/v1/users
GET     /api/v1/users/:id
PATCH   /api/v1/users/:id
DELETE  /api/v1/users/:id

GET     /api/v1/videos
POST    /api/v1/videos               # multipart（ローカル）/ JSON（本番 Blob 登録）
GET     /api/v1/videos/:id
PATCH   /api/v1/videos/:id
DELETE  /api/v1/videos/:id
GET     /api/v1/me/videos
GET     /api/v1/videos/:id/stream

POST    /api/v1/uploads/token        # 大容量アップロード用トークン発行
POST    /api/v1/uploads/blob         # @vercel/blob クライアントトークン窓口

GET     /api/v1/videos/:id/comments
POST    /api/v1/videos/:id/comments
DELETE  /api/v1/comments/:id

GET     /api/v1/videos/:id/like
PUT     /api/v1/videos/:id/like
DELETE  /api/v1/videos/:id/like
```

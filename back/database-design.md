# DB設計書

## 1. 設計方針

### 1.1 採用DB

```txt
PostgreSQL
```

### 1.2 設計原則

1. APIに出すIDとDB内部IDを分ける
2. データ整合性は外部キーと制約で守る
3. 削除履歴を残したい主データは論理削除にする
4. 一覧表示で頻繁に使う集計値は `videos` に保持する
5. SQLはrepository層だけに閉じ込める

### 1.3 ID方針

| 用途 | 型 |
| --- | --- |
| 内部結合 | `BIGSERIAL` |
| 外部公開 | `UUID` |

---

## 2. 概念モデル

```txt
users 1 --- n sessions
users 1 --- n videos
users 1 --- n comments
users 1 --- n video_likes

videos 1 --- n comments
videos 1 --- n video_likes
```

---

## 3. テーブル一覧

| テーブル | 役割 |
| --- | --- |
| `users` | 生徒・先生アカウント |
| `sessions` | ログインセッション |
| `videos` | 動画メタデータ |
| `comments` | コメント |
| `video_likes` | いいね |

---

## 4. Enum / 制約値

### 4.1 ユーザーロール

```sql
CREATE TYPE user_role AS ENUM ('student', 'teacher');
```

### 4.2 動画状態

```sql
CREATE TYPE video_status AS ENUM ('published', 'hidden');
```

---

## 5. テーブル定義

### 5.1 `users`

| カラム | 型 | Null | 説明 |
| --- | --- | --- | --- |
| `id` | `BIGSERIAL` | 不可 | 内部主キー |
| `public_id` | `UUID` | 不可 | API公開ID |
| `username` | `VARCHAR(32)` | 不可 | ログインID |
| `display_name` | `VARCHAR(40)` | 不可 | 表示名 |
| `password_hash` | `TEXT` | 不可 | ハッシュ済みパスワード |
| `role` | `user_role` | 不可 | `student` / `teacher` |
| `is_active` | `BOOLEAN` | 不可 | ログイン許可 |
| `created_at` | `TIMESTAMPTZ` | 不可 | 作成日時 |
| `updated_at` | `TIMESTAMPTZ` | 不可 | 更新日時 |
| `deleted_at` | `TIMESTAMPTZ` | 可 | 論理削除日時 |

#### 主な制約

- `public_id` 一意
- `username` 一意
- `username` は3〜32文字
- `display_name` は空文字不可

### 5.2 `sessions`

| カラム | 型 | Null | 説明 |
| --- | --- | --- | --- |
| `id` | `BIGSERIAL` | 不可 | 内部主キー |
| `user_id` | `BIGINT` | 不可 | ユーザー参照 |
| `token_hash` | `TEXT` | 不可 | セッションIDのハッシュ |
| `expires_at` | `TIMESTAMPTZ` | 不可 | 期限 |
| `created_at` | `TIMESTAMPTZ` | 不可 | 作成日時 |
| `last_seen_at` | `TIMESTAMPTZ` | 不可 | 最終利用日時 |

#### 主な制約

- `token_hash` 一意
- `user_id` は `users.id` を参照

### 5.3 `videos`

| カラム | 型 | Null | 説明 |
| --- | --- | --- | --- |
| `id` | `BIGSERIAL` | 不可 | 内部主キー |
| `public_id` | `UUID` | 不可 | API公開ID |
| `uploader_id` | `BIGINT` | 不可 | 投稿者 |
| `title` | `VARCHAR(80)` | 不可 | タイトル |
| `description` | `VARCHAR(1000)` | 不可 | 説明 |
| `status` | `video_status` | 不可 | 公開状態 |
| `storage_key` | `TEXT` | 不可 | 相対保存キー |
| `original_filename` | `VARCHAR(255)` | 不可 | 元ファイル名 |
| `mime_type` | `VARCHAR(100)` | 不可 | MIME型 |
| `file_size_bytes` | `BIGINT` | 不可 | サイズ |
| `duration_seconds` | `INTEGER` | 可 | 再生秒数 |
| `view_count` | `BIGINT` | 不可 | 再生回数 |
| `like_count` | `BIGINT` | 不可 | いいね数 |
| `comment_count` | `BIGINT` | 不可 | コメント数 |
| `created_at` | `TIMESTAMPTZ` | 不可 | 作成日時 |
| `updated_at` | `TIMESTAMPTZ` | 不可 | 更新日時 |
| `deleted_at` | `TIMESTAMPTZ` | 可 | 論理削除日時 |

#### 主な制約

- `public_id` 一意
- `storage_key` 一意
- `file_size_bytes > 0`
- 集計値は0以上
- `uploader_id` は `users.id` を参照

### 5.4 `comments`

| カラム | 型 | Null | 説明 |
| --- | --- | --- | --- |
| `id` | `BIGSERIAL` | 不可 | 内部主キー |
| `public_id` | `UUID` | 不可 | API公開ID |
| `video_id` | `BIGINT` | 不可 | 動画参照 |
| `user_id` | `BIGINT` | 不可 | 投稿者 |
| `body` | `VARCHAR(300)` | 不可 | コメント本文 |
| `created_at` | `TIMESTAMPTZ` | 不可 | 作成日時 |
| `updated_at` | `TIMESTAMPTZ` | 不可 | 更新日時 |
| `deleted_at` | `TIMESTAMPTZ` | 可 | 論理削除日時 |

#### 主な制約

- `public_id` 一意
- `video_id` は `videos.id` を参照
- `user_id` は `users.id` を参照
- `body` は空文字不可

### 5.5 `video_likes`

| カラム | 型 | Null | 説明 |
| --- | --- | --- | --- |
| `video_id` | `BIGINT` | 不可 | 動画参照 |
| `user_id` | `BIGINT` | 不可 | ユーザー参照 |
| `created_at` | `TIMESTAMPTZ` | 不可 | 作成日時 |

#### 主な制約

- 主キーは `(video_id, user_id)`
- 同一ユーザーは同一動画に1回だけいいねできる

---

## 6. 推奨DDL

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE user_role AS ENUM ('student', 'teacher');
CREATE TYPE video_status AS ENUM ('published', 'hidden');

CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  public_id UUID NOT NULL DEFAULT gen_random_uuid(),
  username VARCHAR(32) NOT NULL,
  display_name VARCHAR(40) NOT NULL,
  password_hash TEXT NOT NULL,
  role user_role NOT NULL DEFAULT 'student',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT users_public_id_unique UNIQUE (public_id),
  CONSTRAINT users_username_unique UNIQUE (username),
  CONSTRAINT users_username_length CHECK (char_length(username) BETWEEN 3 AND 32),
  CONSTRAINT users_display_name_not_blank CHECK (char_length(btrim(display_name)) > 0)
);

CREATE TABLE sessions (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT sessions_token_hash_unique UNIQUE (token_hash)
);

CREATE TABLE videos (
  id BIGSERIAL PRIMARY KEY,
  public_id UUID NOT NULL DEFAULT gen_random_uuid(),
  uploader_id BIGINT NOT NULL REFERENCES users(id),
  title VARCHAR(80) NOT NULL,
  description VARCHAR(1000) NOT NULL DEFAULT '',
  status video_status NOT NULL DEFAULT 'published',
  storage_key TEXT NOT NULL,
  original_filename VARCHAR(255) NOT NULL,
  mime_type VARCHAR(100) NOT NULL,
  file_size_bytes BIGINT NOT NULL,
  duration_seconds INTEGER,
  view_count BIGINT NOT NULL DEFAULT 0,
  like_count BIGINT NOT NULL DEFAULT 0,
  comment_count BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT videos_public_id_unique UNIQUE (public_id),
  CONSTRAINT videos_storage_key_unique UNIQUE (storage_key),
  CONSTRAINT videos_title_not_blank CHECK (char_length(btrim(title)) > 0),
  CONSTRAINT videos_file_size_positive CHECK (file_size_bytes > 0),
  CONSTRAINT videos_duration_non_negative CHECK (duration_seconds IS NULL OR duration_seconds >= 0),
  CONSTRAINT videos_view_count_non_negative CHECK (view_count >= 0),
  CONSTRAINT videos_like_count_non_negative CHECK (like_count >= 0),
  CONSTRAINT videos_comment_count_non_negative CHECK (comment_count >= 0)
);

CREATE TABLE comments (
  id BIGSERIAL PRIMARY KEY,
  public_id UUID NOT NULL DEFAULT gen_random_uuid(),
  video_id BIGINT NOT NULL REFERENCES videos(id),
  user_id BIGINT NOT NULL REFERENCES users(id),
  body VARCHAR(300) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT comments_public_id_unique UNIQUE (public_id),
  CONSTRAINT comments_body_not_blank CHECK (char_length(btrim(body)) > 0)
);

CREATE TABLE video_likes (
  video_id BIGINT NOT NULL REFERENCES videos(id),
  user_id BIGINT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (video_id, user_id)
);
```

---

## 7. インデックス

```sql
CREATE INDEX idx_users_role_active
  ON users (role, is_active)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_sessions_user_id
  ON sessions (user_id);

CREATE INDEX idx_sessions_expires_at
  ON sessions (expires_at);

CREATE INDEX idx_videos_status_created_at
  ON videos (status, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_videos_uploader_created_at
  ON videos (uploader_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_videos_created_at
  ON videos (created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_comments_video_created_at
  ON comments (video_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_comments_user_created_at
  ON comments (user_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_video_likes_user_id
  ON video_likes (user_id);
```

### 7.1 検索用インデックスの扱い

初期版は小規模運用を想定し、`ILIKE` 検索で十分とする。  
動画数が増えて検索が遅くなった場合だけ、`pg_trgm` を使う。

```sql
-- 将来必要になった時だけ追加
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_videos_title_trgm
  ON videos USING gin (title gin_trgm_ops);
```

---

## 8. 論理削除ポリシー

| テーブル | 方針 |
| --- | --- |
| `users` | 論理削除 |
| `videos` | 論理削除 |
| `comments` | 論理削除 |
| `sessions` | 物理削除 |
| `video_likes` | 物理削除 |

### 8.1 理由

- ユーザー、動画、コメントは授業中の事故復旧をしやすくする
- セッションといいねは履歴より現在状態が大事なので物理削除でよい

---

## 9. 集計カラム

### 9.1 `videos.view_count`

動画再生時に増やす。

### 9.2 `videos.like_count`

`video_likes` の追加/削除と同じトランザクションで更新する。

### 9.3 `videos.comment_count`

`comments` の追加/削除と同じトランザクションで更新する。

### 9.4 なぜ持つか

一覧画面で毎回 `COUNT(*)` を実行すると説明が複雑になり、表示も重くなる。  
教育用途では、トランザクションと集計値の関係を学ぶ題材にもなる。

---

## 10. ファイル保存設計

### 10.1 保存先

```txt
server/uploads/videos/YYYY/MM/<uuid>.<ext>
```

### 10.2 DBに保存する値

```txt
videos.storage_key = videos/2026/05/1f1d3fb8-5b54-48b0-83c5-d5f5c52d82e9.mp4
```

### 10.3 ルール

- 絶対パスは保存しない
- 元ファイル名は `original_filename` に分ける
- 保存キーはサーバー側で生成する
- 途中失敗時はファイルを残さない

---

## 11. マイグレーション順序

1. 拡張機能作成
2. enum作成
3. `users`
4. `sessions`
5. `videos`
6. `comments`
7. `video_likes`
8. インデックス
9. 初期先生ユーザー投入

---

## 12. 初期データ

最低限、先生ユーザーを1件作る。

```sql
INSERT INTO users (
  username,
  display_name,
  password_hash,
  role
) VALUES (
  'teacher',
  '先生',
  '$2a$12$replace_with_real_hash',
  'teacher'
);
```

平文パスワードは絶対にDBへ保存しない。

---

## 13. 更新日時の扱い

初期版では、`updated_at` はアプリケーション側で更新する。  
トリガーを使えば自動化できるが、教育用途では「更新時に何が変わるか」をservice層から追いやすい方がよい。

---

## 14. 代表的な削除時の影響

### 14.1 ユーザー削除

- `users.deleted_at` を埋める
- 既存動画やコメントは残す
- `sessions` は即時削除
- 画面表示では表示名を残してもよい

### 14.2 動画削除

- `videos.deleted_at` を埋める
- 動画ファイルは削除
- コメントといいねは一覧から見えなくなる

### 14.3 コメント削除

- `comments.deleted_at` を埋める
- `videos.comment_count` を減らす

---

## 15. 将来拡張候補

必要になった時だけ追加を検討する。

| 機能 | 追加候補 |
| --- | --- |
| タグ | `tags`, `video_tags` |
| サムネイル | `thumbnail_storage_key` |
| 再生履歴 | `video_views` |
| 教材分類 | `classes`, `class_memberships` |
| 通報 | `reports` |

初期版ではテーブルを増やしすぎず、学習しやすい最小構成を保つ。

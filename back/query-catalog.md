# クエリ設計書

## 1. 目的

この資料は、repository層に置くSQLの候補を一覧化したものである。  
handlerやserviceがSQLを直接書かないよう、あらかじめ用途ごとのクエリを分ける。

---

## 2. 命名方針

| repository | 代表メソッド |
| --- | --- |
| `UserRepository` | `Create`, `FindByUsername`, `FindByPublicID`, `List`, `Update`, `SoftDelete` |
| `SessionRepository` | `Create`, `FindActiveByTokenHash`, `Touch`, `DeleteByTokenHash`, `DeleteExpired` |
| `VideoRepository` | `Create`, `FindByPublicID`, `ListPublished`, `ListByUploader`, `UpdateMetadata`, `UpdateStatus`, `SoftDelete`, `IncrementViewCount` |
| `CommentRepository` | `Create`, `ListByVideo`, `FindByPublicID`, `SoftDelete` |
| `LikeRepository` | `Exists`, `Create`, `Delete` |

---

## 3. ユーザー系クエリ

### 3.1 `CreateUser`

```sql
INSERT INTO users (
  username,
  display_name,
  password_hash,
  role
) VALUES (
  $1, $2, $3, $4
)
RETURNING id, public_id, username, display_name, role, is_active, created_at, updated_at;
```

### 3.2 `FindUserByUsername`

```sql
SELECT
  id,
  public_id,
  username,
  display_name,
  password_hash,
  role,
  is_active,
  created_at,
  updated_at
FROM users
WHERE username = $1
  AND deleted_at IS NULL;
```

### 3.3 `FindUserByPublicID`

```sql
SELECT
  id,
  public_id,
  username,
  display_name,
  role,
  is_active,
  created_at,
  updated_at
FROM users
WHERE public_id = $1
  AND deleted_at IS NULL;
```

### 3.4 `ListUsers`

```sql
SELECT
  public_id,
  username,
  display_name,
  role,
  is_active,
  created_at,
  updated_at
FROM users
WHERE deleted_at IS NULL
  AND ($1::user_role IS NULL OR role = $1)
  AND ($2::BOOLEAN IS NULL OR is_active = $2)
  AND (
    $3::TEXT IS NULL
    OR username ILIKE '%' || $3 || '%'
    OR display_name ILIKE '%' || $3 || '%'
  )
ORDER BY created_at DESC
LIMIT $4
OFFSET $5;
```

### 3.5 `CountUsers`

```sql
SELECT COUNT(*)
FROM users
WHERE deleted_at IS NULL
  AND ($1::user_role IS NULL OR role = $1)
  AND ($2::BOOLEAN IS NULL OR is_active = $2)
  AND (
    $3::TEXT IS NULL
    OR username ILIKE '%' || $3 || '%'
    OR display_name ILIKE '%' || $3 || '%'
  );
```

### 3.6 `UpdateUser`

```sql
UPDATE users
SET
  display_name = COALESCE($2, display_name),
  password_hash = COALESCE($3, password_hash),
  role = COALESCE($4, role),
  is_active = COALESCE($5, is_active),
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL
RETURNING public_id, username, display_name, role, is_active, created_at, updated_at;
```

### 3.7 `SoftDeleteUser`

```sql
UPDATE users
SET
  deleted_at = NOW(),
  is_active = FALSE,
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL;
```

---

## 4. セッション系クエリ

### 4.1 `CreateSession`

```sql
INSERT INTO sessions (
  user_id,
  token_hash,
  expires_at
) VALUES (
  $1, $2, $3
)
RETURNING id, user_id, expires_at, created_at, last_seen_at;
```

### 4.2 `FindActiveSessionWithUser`

```sql
SELECT
  s.id AS session_id,
  s.user_id,
  s.expires_at,
  u.public_id,
  u.username,
  u.display_name,
  u.role,
  u.is_active
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND s.expires_at > NOW()
  AND u.deleted_at IS NULL
  AND u.is_active = TRUE;
```

### 4.3 `TouchSession`

```sql
UPDATE sessions
SET last_seen_at = NOW()
WHERE id = $1;
```

### 4.4 `DeleteSessionByTokenHash`

```sql
DELETE FROM sessions
WHERE token_hash = $1;
```

### 4.5 `DeleteSessionsByUserID`

```sql
DELETE FROM sessions
WHERE user_id = $1;
```

### 4.6 `DeleteExpiredSessions`

```sql
DELETE FROM sessions
WHERE expires_at <= NOW();
```

---

## 5. 動画系クエリ

### 5.1 `CreateVideo`

```sql
INSERT INTO videos (
  uploader_id,
  title,
  description,
  storage_key,
  original_filename,
  mime_type,
  file_size_bytes,
  duration_seconds
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING
  id,
  public_id,
  uploader_id,
  title,
  description,
  status,
  storage_key,
  original_filename,
  mime_type,
  file_size_bytes,
  duration_seconds,
  view_count,
  like_count,
  comment_count,
  created_at,
  updated_at;
```

### 5.2 `FindVideoByPublicID`

```sql
SELECT
  v.id,
  v.public_id,
  v.title,
  v.description,
  v.status,
  v.mime_type,
  v.file_size_bytes,
  v.duration_seconds,
  v.view_count,
  v.like_count,
  v.comment_count,
  v.created_at,
  v.updated_at,
  u.public_id AS uploader_public_id,
  u.display_name AS uploader_display_name
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.public_id = $1
  AND v.deleted_at IS NULL;
```

### 5.3 `FindVideoStorageInfo`

```sql
SELECT
  id,
  public_id,
  uploader_id,
  status,
  storage_key,
  mime_type,
  file_size_bytes,
  deleted_at
FROM videos
WHERE public_id = $1
  AND deleted_at IS NULL;
```

### 5.4 `ListPublishedVideos`

```sql
SELECT
  v.public_id,
  v.title,
  v.description,
  v.status,
  v.mime_type,
  v.file_size_bytes,
  v.duration_seconds,
  v.view_count,
  v.like_count,
  v.comment_count,
  v.created_at,
  v.updated_at,
  u.public_id AS uploader_public_id,
  u.display_name AS uploader_display_name
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.deleted_at IS NULL
  AND v.status = 'published'
  AND (
    $1::TEXT IS NULL
    OR v.title ILIKE '%' || $1 || '%'
    OR v.description ILIKE '%' || $1 || '%'
    OR u.display_name ILIKE '%' || $1 || '%'
  )
  AND ($2::UUID IS NULL OR u.public_id = $2)
ORDER BY v.created_at DESC
LIMIT $3
OFFSET $4;
```

### 5.5 `CountPublishedVideos`

```sql
SELECT COUNT(*)
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.deleted_at IS NULL
  AND v.status = 'published'
  AND (
    $1::TEXT IS NULL
    OR v.title ILIKE '%' || $1 || '%'
    OR v.description ILIKE '%' || $1 || '%'
    OR u.display_name ILIKE '%' || $1 || '%'
  )
  AND ($2::UUID IS NULL OR u.public_id = $2);
```

### 5.6 `ListVideosForTeacher`

```sql
SELECT
  v.public_id,
  v.title,
  v.description,
  v.status,
  v.mime_type,
  v.file_size_bytes,
  v.duration_seconds,
  v.view_count,
  v.like_count,
  v.comment_count,
  v.created_at,
  v.updated_at,
  u.public_id AS uploader_public_id,
  u.display_name AS uploader_display_name
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.deleted_at IS NULL
  AND ($1::video_status IS NULL OR v.status = $1)
  AND (
    $2::TEXT IS NULL
    OR v.title ILIKE '%' || $2 || '%'
    OR v.description ILIKE '%' || $2 || '%'
    OR u.display_name ILIKE '%' || $2 || '%'
  )
ORDER BY v.created_at DESC
LIMIT $3
OFFSET $4;
```

### 5.7 `CountVideosForTeacher`

```sql
SELECT COUNT(*)
FROM videos v
JOIN users u ON u.id = v.uploader_id
WHERE v.deleted_at IS NULL
  AND ($1::video_status IS NULL OR v.status = $1)
  AND (
    $2::TEXT IS NULL
    OR v.title ILIKE '%' || $2 || '%'
    OR v.description ILIKE '%' || $2 || '%'
    OR u.display_name ILIKE '%' || $2 || '%'
  );
```

### 5.8 `ListVideosByUploader`

```sql
SELECT
  public_id,
  title,
  description,
  status,
  mime_type,
  file_size_bytes,
  duration_seconds,
  view_count,
  like_count,
  comment_count,
  created_at,
  updated_at
FROM videos
WHERE uploader_id = $1
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;
```

### 5.9 `CountVideosByUploader`

```sql
SELECT COUNT(*)
FROM videos
WHERE uploader_id = $1
  AND deleted_at IS NULL;
```

### 5.10 `UpdateVideoMetadata`

```sql
UPDATE videos
SET
  title = COALESCE($2, title),
  description = COALESCE($3, description),
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL
RETURNING
  public_id,
  title,
  description,
  status,
  updated_at;
```

### 5.11 `UpdateVideoStatus`

```sql
UPDATE videos
SET
  status = $2,
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL
RETURNING public_id, status, updated_at;
```

### 5.12 `SoftDeleteVideo`

```sql
UPDATE videos
SET
  deleted_at = NOW(),
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL
RETURNING storage_key;
```

### 5.13 `IncrementVideoViewCount`

```sql
UPDATE videos
SET view_count = view_count + 1
WHERE id = $1
  AND deleted_at IS NULL;
```

---

## 6. コメント系クエリ

### 6.1 `CreateComment`

```sql
INSERT INTO comments (
  video_id,
  user_id,
  body
) VALUES (
  $1, $2, $3
)
RETURNING id, public_id, video_id, user_id, body, created_at, updated_at;
```

### 6.2 `ListCommentsByVideo`

```sql
SELECT
  c.public_id,
  c.body,
  c.created_at,
  c.updated_at,
  u.public_id AS author_public_id,
  u.display_name AS author_display_name
FROM comments c
JOIN users u ON u.id = c.user_id
WHERE c.video_id = $1
  AND c.deleted_at IS NULL
ORDER BY c.created_at DESC
LIMIT $2
OFFSET $3;
```

### 6.3 `CountCommentsByVideo`

```sql
SELECT COUNT(*)
FROM comments
WHERE video_id = $1
  AND deleted_at IS NULL;
```

### 6.4 `FindCommentByPublicID`

```sql
SELECT
  id,
  public_id,
  video_id,
  user_id,
  body,
  deleted_at
FROM comments
WHERE public_id = $1;
```

### 6.5 `SoftDeleteComment`

```sql
UPDATE comments
SET
  deleted_at = NOW(),
  updated_at = NOW()
WHERE public_id = $1
  AND deleted_at IS NULL
RETURNING video_id;
```

---

## 7. いいね系クエリ

### 7.1 `FindLikeState`

```sql
SELECT EXISTS (
  SELECT 1
  FROM video_likes
  WHERE video_id = $1
    AND user_id = $2
) AS liked;
```

### 7.2 `CreateLike`

```sql
INSERT INTO video_likes (
  video_id,
  user_id
) VALUES (
  $1, $2
)
ON CONFLICT (video_id, user_id) DO NOTHING
RETURNING video_id;
```

### 7.3 `DeleteLike`

```sql
DELETE FROM video_likes
WHERE video_id = $1
  AND user_id = $2
RETURNING video_id;
```

---

## 8. 集計更新クエリ

### 8.1 `IncrementVideoLikeCount`

```sql
UPDATE videos
SET like_count = like_count + 1
WHERE id = $1
  AND deleted_at IS NULL;
```

### 8.2 `DecrementVideoLikeCount`

```sql
UPDATE videos
SET like_count = GREATEST(like_count - 1, 0)
WHERE id = $1
  AND deleted_at IS NULL;
```

### 8.3 `IncrementVideoCommentCount`

```sql
UPDATE videos
SET comment_count = comment_count + 1
WHERE id = $1
  AND deleted_at IS NULL;
```

### 8.4 `DecrementVideoCommentCount`

```sql
UPDATE videos
SET comment_count = GREATEST(comment_count - 1, 0)
WHERE id = $1
  AND deleted_at IS NULL;
```

---

## 9. 代表的なトランザクション

### 9.1 コメント作成

```sql
BEGIN;

INSERT INTO comments (
  video_id,
  user_id,
  body
) VALUES (
  $1, $2, $3
);

UPDATE videos
SET comment_count = comment_count + 1
WHERE id = $1
  AND deleted_at IS NULL;

COMMIT;
```

### 9.2 コメント削除

```sql
BEGIN;

WITH deleted_comment AS (
  UPDATE comments
  SET
    deleted_at = NOW(),
    updated_at = NOW()
  WHERE public_id = $1
    AND deleted_at IS NULL
  RETURNING video_id
)
UPDATE videos
SET comment_count = GREATEST(comment_count - 1, 0)
WHERE id IN (SELECT video_id FROM deleted_comment);

COMMIT;
```

### 9.3 いいね追加

```sql
BEGIN;

WITH inserted_like AS (
  INSERT INTO video_likes (
    video_id,
    user_id
  ) VALUES (
    $1, $2
  )
  ON CONFLICT (video_id, user_id) DO NOTHING
  RETURNING video_id
)
UPDATE videos
SET like_count = like_count + 1
WHERE id IN (SELECT video_id FROM inserted_like);

COMMIT;
```

### 9.4 いいね解除

```sql
BEGIN;

WITH deleted_like AS (
  DELETE FROM video_likes
  WHERE video_id = $1
    AND user_id = $2
  RETURNING video_id
)
UPDATE videos
SET like_count = GREATEST(like_count - 1, 0)
WHERE id IN (SELECT video_id FROM deleted_like);

COMMIT;
```

---

## 10. 保守用クエリ

### 10.1 期限切れセッション削除

```sql
DELETE FROM sessions
WHERE expires_at <= NOW();
```

### 10.2 孤立動画確認

DBに存在するのにファイルが無い動画はアプリ側でファイル存在確認が必要なので、まず候補だけ取得する。

```sql
SELECT
  public_id,
  storage_key
FROM videos
WHERE deleted_at IS NULL;
```

### 10.3 集計再計算

もし何らかの障害で集計値がずれた時の修復用。

```sql
UPDATE videos v
SET like_count = counts.like_count
FROM (
  SELECT
    video_id,
    COUNT(*) AS like_count
  FROM video_likes
  GROUP BY video_id
) counts
WHERE v.id = counts.video_id;
```

```sql
UPDATE videos v
SET comment_count = counts.comment_count
FROM (
  SELECT
    video_id,
    COUNT(*) AS comment_count
  FROM comments
  WHERE deleted_at IS NULL
  GROUP BY video_id
) counts
WHERE v.id = counts.video_id;
```

---

## 11. 実装時の注意

### 11.1 `UPDATE ... RETURNING`

更新直後の値を返せるため、余計な再SELECTを減らせる。

### 11.2 一覧と件数取得

ページング用の一覧クエリと件数クエリは条件をそろえる。  
片方だけ条件が違うと、表示件数とページ数がずれる。

### 11.3 動画削除とファイル削除

DB更新とファイル削除は同一トランザクションにできない。  
実装では次の順が分かりやすい。

1. DBから対象動画と `storage_key` を取得
2. 権限確認
3. DBを論理削除
4. ファイルを削除
5. ファイル削除に失敗したらログへ記録

### 11.4 アップロード失敗時

ファイル保存後にDB INSERTが失敗した場合は、service層で保存済みファイルを削除する。

### 11.5 先生向け一覧

先生一覧と生徒向け一覧はSQLを分ける。  
1本の巨大クエリで全権限を処理しようとすると、説明しにくくミスも増える。

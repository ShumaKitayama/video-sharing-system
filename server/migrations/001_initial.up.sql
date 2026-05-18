CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$ BEGIN
  CREATE TYPE user_role AS ENUM ('student', 'teacher');
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  CREATE TYPE video_status AS ENUM ('published', 'hidden');
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS users (
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

CREATE TABLE IF NOT EXISTS sessions (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id),
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT sessions_token_hash_unique UNIQUE (token_hash)
);

CREATE TABLE IF NOT EXISTS videos (
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

CREATE TABLE IF NOT EXISTS comments (
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

CREATE TABLE IF NOT EXISTS video_likes (
  video_id BIGINT NOT NULL REFERENCES videos(id),
  user_id BIGINT NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (video_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_users_role_active
  ON users (role, is_active)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sessions_user_id
  ON sessions (user_id);

CREATE INDEX IF NOT EXISTS idx_sessions_expires_at
  ON sessions (expires_at);

CREATE INDEX IF NOT EXISTS idx_videos_status_created_at
  ON videos (status, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_videos_uploader_created_at
  ON videos (uploader_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_videos_created_at
  ON videos (created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comments_video_created_at
  ON comments (video_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comments_user_created_at
  ON comments (user_id, created_at DESC)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_video_likes_user_id
  ON video_likes (user_id);

INSERT INTO users (
  username,
  display_name,
  password_hash,
  role
) VALUES (
  'teacher',
  '先生',
  '$2a$12$U3AfXFFBMzIC/v1I4xMdxezRACmdRlQ4o6NTRTjpmFm5NZ/uJPswC',
  'teacher'
)
ON CONFLICT (username) DO NOTHING;

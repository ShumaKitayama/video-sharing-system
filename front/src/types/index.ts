/* ===================================================
 * types/index.ts
 * システム全体で共有するデータ型の定義
 * =================================================== */

/** ユーザー情報（APIレスポンスの形） */
export interface User {
  id: string;
  username: string;
  display_name: string;
  role: "student" | "teacher";
}

/** 動画の投稿者（埋め込み表現） */
export interface VideoUploader {
  id: string;
  display_name: string;
}

/** 動画メタデータ */
export interface Video {
  id: string;
  title: string;
  description: string;
  status: "published" | "hidden";
  uploader: VideoUploader;
  mime_type: string;
  file_size_bytes: number;
  duration_seconds: number | null;
  view_count: number;
  like_count: number;
  comment_count: number;
  created_at: string;
  updated_at: string;
  /** フロントエンド表示用のサムネイルURL（将来拡張） */
  thumbnail_url?: string;
}

/** コメントの投稿者（埋め込み表現） */
export interface CommentAuthor {
  id: string;
  display_name: string;
}

/** コメント */
export interface Comment {
  id: string;
  body: string;
  author: CommentAuthor;
  created_at: string;
  updated_at: string;
}

/** ページネーション情報 */
export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

/** 一覧APIレスポンスの共通形 */
export interface PaginatedResponse<T> {
  data: T[];
  meta: PaginationMeta;
}

/** 動画検索・一覧のクエリパラメータ */
export interface VideoQueryParams {
  page?: number;
  per_page?: number;
  q?: string;
  sort?: "latest" | "oldest" | "most_viewed" | "most_liked";
  uploader_id?: string;
}

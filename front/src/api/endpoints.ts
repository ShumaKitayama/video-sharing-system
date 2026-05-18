/* ===================================================
 * api/endpoints.ts
 * [保護] APIエンドポイントの定義
 * バックエンドのベースURLとパスを一元管理する
 * =================================================== */

/** APIベースURL（環境変数から取得、デフォルトはローカル開発用） */
export const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

/** APIバージョンプレフィックス */
const V1 = "/api/v1";

/** エンドポイント定義 */
export const endpoints = {
  // 認証
  auth: {
    login: `${V1}/auth/login`,
    logout: `${V1}/auth/logout`,
    me: `${V1}/auth/me`,
  },

  // 動画
  videos: {
    list: `${V1}/videos`,
    detail: (id: string) => `${V1}/videos/${id}`,
    create: `${V1}/videos`,
    update: (id: string) => `${V1}/videos/${id}`,
    delete: (id: string) => `${V1}/videos/${id}`,
    myVideos: `${V1}/me/videos`,
    stream: (id: string) => `${V1}/videos/${id}/stream`,
  },

  // コメント
  comments: {
    list: (videoId: string) => `${V1}/videos/${videoId}/comments`,
    create: (videoId: string) => `${V1}/videos/${videoId}/comments`,
    delete: (commentId: string) => `${V1}/comments/${commentId}`,
  },

  // いいね
  likes: {
    status: (videoId: string) => `${V1}/videos/${videoId}/like`,
    add: (videoId: string) => `${V1}/videos/${videoId}/like`,
    remove: (videoId: string) => `${V1}/videos/${videoId}/like`,
  },

  // ユーザー
  users: {
    list: `${V1}/users`,
    detail: (id: string) => `${V1}/users/${id}`,
  },
} as const;

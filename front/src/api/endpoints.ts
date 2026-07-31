/* ===================================================
 * api/endpoints.ts
 * [保護] APIエンドポイントの定義
 * バックエンドのベースURLとパスを一元管理する
 * =================================================== */

/**
 * リモート（Vercel）のバックエンドに接続するモードか。
 * Docker（front/Dockerfile.dev）で起動したときに true になる。
 */
export const USE_REMOTE_BACKEND =
  import.meta.env.VITE_REMOTE_BACKEND === "true";

/** APIベースURL（環境変数から取得、デフォルトはローカル開発用） */
function resolveApiBaseUrl(): string {
  const fromEnv = import.meta.env.VITE_API_BASE_URL;
  if (typeof fromEnv === "string") {
    return fromEnv;
  }
  // Docker 開発: Vite の dev proxy 経由で同一オリジンに API を中継
  if (USE_REMOTE_BACKEND) {
    return "";
  }
  // Vercel 本番: フロント vercel.json の rewrite 経由で同一オリジンに API を中継
  return import.meta.env.PROD ? "" : "http://localhost:8080";
}

export const API_BASE_URL = resolveApiBaseUrl();

/** 大容量アップロード用の API 直 URL（Cookie ではなくトークン認証） */
function resolveDirectApiBaseUrl(): string {
  const fromEnv = import.meta.env.VITE_API_DIRECT_URL;
  if (typeof fromEnv === "string" && fromEnv) {
    return fromEnv;
  }
  // Docker 開発: ローカルの API は無いので dev proxy 経由に揃える
  if (USE_REMOTE_BACKEND) {
    return "";
  }
  return import.meta.env.PROD
    ? "https://video-sharing-api.vercel.app"
    : "http://localhost:8080";
}

export const API_DIRECT_BASE_URL = resolveDirectApiBaseUrl();

/** APIバージョンプレフィックス */
const V1 = "/api/v1";

/** エンドポイント定義 */
export const endpoints = {
  // 認証
  auth: {
    register: `${V1}/auth/register`,
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

  // アップロード
  uploads: {
    token: `${V1}/uploads/token`,
    blob: `${V1}/uploads/blob`,
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

/* ===================================================
 * services/api.ts
 * [保護] API通信サービス層
 * バックエンドの実エンドポイントを呼び出す。
 * フック層（hooks/）はこのサービスだけを使い、
 * fetch を直接書かない。
 * =================================================== */

import type { Video, Comment, VideoQueryParams, PaginatedResponse } from "../types";
import { API_BASE_URL, endpoints } from "../api/endpoints";

// -------------------------------------------------------
// 共通エラークラス
// -------------------------------------------------------

/** APIエラー情報を保持するクラス */
export class ApiError extends Error {
  status: number;
  code: string;
  details?: { field: string; message: string }[];

  constructor(
    status: number,
    code: string,
    details?: { field: string; message: string }[],
  ) {
    super(code);
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

// -------------------------------------------------------
// 共通 fetch ラッパー
// -------------------------------------------------------

/**
 * 全APIリクエストを通る共通関数。
 * - credentials: "include" でCookieを自動送信
 * - 204 No Content は undefined を返す
 * - エラー時は ApiError をスローする
 */
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

// -------------------------------------------------------
// 認証サービス
// -------------------------------------------------------

export interface RegisterInput {
  username: string;
  display_name: string;
  password: string;
}

export interface AuthUserPayload {
  id: string;
  username: string;
  display_name: string;
  role: "student" | "teacher";
}

export const authService = {
  /** 新規アカウント登録（生徒ロール固定） */
  async register(input: RegisterInput): Promise<AuthUserPayload> {
    const res = await apiFetch<{ user: AuthUserPayload }>(
      endpoints.auth.register,
      {
        method: "POST",
        body: JSON.stringify(input),
      },
    );
    return res.user;
  },
};

// -------------------------------------------------------
// 動画サービス
// -------------------------------------------------------

export const videoService = {
  /** 動画一覧を取得する（ページング・検索・ソート対応） */
  async getVideos(params?: VideoQueryParams): Promise<PaginatedResponse<Video>> {
    const query = new URLSearchParams();
    if (params?.page) query.set("page", String(params.page));
    if (params?.per_page) query.set("per_page", String(params.per_page));
    if (params?.q) query.set("q", params.q);
    if (params?.sort) query.set("sort", params.sort);
    if (params?.uploader_id) query.set("uploader_id", params.uploader_id);

    const qs = query.toString();
    const url = `${endpoints.videos.list}${qs ? `?${qs}` : ""}`;

    const res = await fetch(`${API_BASE_URL}${url}`, {
      credentials: "include",
      headers: { "Content-Type": "application/json" },
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new ApiError(
        res.status,
        body.error?.code ?? "INTERNAL_ERROR",
        body.error?.details,
      );
    }

    const body = await res.json();
    // バックエンドは { data: [...], meta: {...} } 形式で返す
    return {
      data: body.data as Video[],
      meta: body.meta,
    };
  },

  /** 動画詳細を取得する */
  async getVideoById(id: string): Promise<Video | null> {
    try {
      return await apiFetch<Video>(endpoints.videos.detail(id));
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) return null;
      throw e;
    }
  },

  /** 自分の動画一覧を取得する（ログイン必須） */
  async getMyVideos(): Promise<PaginatedResponse<Video>> {
    const res = await fetch(`${API_BASE_URL}${endpoints.videos.myVideos}`, {
      credentials: "include",
      headers: { "Content-Type": "application/json" },
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new ApiError(res.status, body.error?.code ?? "INTERNAL_ERROR");
    }

    const body = await res.json();
    return { data: body.data as Video[], meta: body.meta };
  },

  /** 動画を削除する（ログイン必須、投稿者本人または先生） */
  async deleteVideo(id: string): Promise<void> {
    await apiFetch<void>(endpoints.videos.delete(id), { method: "DELETE" });
  },
};

// -------------------------------------------------------
// コメントサービス
// -------------------------------------------------------

export const commentService = {
  /** コメント一覧を取得する */
  async getComments(videoId: string): Promise<Comment[]> {
    const res = await fetch(
      `${API_BASE_URL}${endpoints.comments.list(videoId)}`,
      { credentials: "include" },
    );
    if (!res.ok) return [];
    const body = await res.json();
    // バックエンドは { data: [...] } 形式
    return (body.data ?? []) as Comment[];
  },

  /** コメントを投稿する（ログイン必須） */
  async postComment(videoId: string, body: string): Promise<Comment> {
    return apiFetch<Comment>(endpoints.comments.create(videoId), {
      method: "POST",
      body: JSON.stringify({ body }),
    });
  },

  /** コメントを削除する（ログイン必須） */
  async deleteComment(commentId: string): Promise<void> {
    await apiFetch<void>(endpoints.comments.delete(commentId), {
      method: "DELETE",
    });
  },
};

// -------------------------------------------------------
// いいねサービス
// -------------------------------------------------------

export const likeService = {
  /** 自分がいいね済みかどうかを取得する（ログイン必須） */
  async getLikeStatus(videoId: string): Promise<boolean> {
    try {
      const res = await fetch(
        `${API_BASE_URL}${endpoints.likes.status(videoId)}`,
        { credentials: "include" },
      );
      if (!res.ok) return false;
      const body = await res.json();
      return (body.data?.liked ?? false) as boolean;
    } catch {
      return false;
    }
  },

  /** いいねを追加する（冪等）。{ liked, like_count } を返す */
  async addLike(videoId: string): Promise<{ liked: boolean; like_count: number }> {
    return apiFetch<{ liked: boolean; like_count: number }>(
      endpoints.likes.add(videoId),
      { method: "PUT" },
    );
  },

  /** いいねを解除する（冪等）。{ liked, like_count } を返す */
  async removeLike(videoId: string): Promise<{ liked: boolean; like_count: number }> {
    return apiFetch<{ liked: boolean; like_count: number }>(
      endpoints.likes.remove(videoId),
      { method: "DELETE" },
    );
  },
};

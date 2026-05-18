/* ===================================================
 * services/api.ts
 * [保護] API通信サービス層
 * 現在はモックデータを返す。将来のバックエンド接続時は
 * このファイルの中身だけを差し替えればフック層以上は無変更で動く。
 * =================================================== */

import type { Video, Comment, VideoQueryParams, PaginatedResponse } from "../types";
import { mockVideos, mockComments } from "../mock/data";

/** 擬似的な通信遅延を再現する */
function delay(ms: number = 200): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/* ---------- 動画サービス ---------- */

export const videoService = {
  /** 動画一覧を取得する */
  async getVideos(params?: VideoQueryParams): Promise<PaginatedResponse<Video>> {
    await delay();

    let filtered = [...mockVideos];

    // 検索
    if (params?.q) {
      const q = params.q.toLowerCase();
      filtered = filtered.filter(
        (v) =>
          v.title.toLowerCase().includes(q) ||
          v.description.toLowerCase().includes(q) ||
          v.uploader.display_name.toLowerCase().includes(q)
      );
    }

    // 投稿者絞り込み
    if (params?.uploader_id) {
      filtered = filtered.filter((v) => v.uploader.id === params.uploader_id);
    }

    // ソート
    switch (params?.sort) {
      case "oldest":
        filtered.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime());
        break;
      case "most_viewed":
        filtered.sort((a, b) => b.view_count - a.view_count);
        break;
      case "most_liked":
        filtered.sort((a, b) => b.like_count - a.like_count);
        break;
      default: // latest
        filtered.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
    }

    const page = params?.page ?? 1;
    const perPage = params?.per_page ?? 20;
    const start = (page - 1) * perPage;
    const paged = filtered.slice(start, start + perPage);

    return {
      data: paged,
      meta: {
        page,
        per_page: perPage,
        total: filtered.length,
        total_pages: Math.ceil(filtered.length / perPage),
      },
    };
  },

  /** 動画詳細を取得する */
  async getVideoById(id: string): Promise<Video | null> {
    await delay();
    return mockVideos.find((v) => v.id === id) ?? null;
  },
};

/* ---------- コメントサービス ---------- */

export const commentService = {
  /** コメント一覧を取得する */
  async getComments(videoId: string): Promise<Comment[]> {
    await delay();
    return mockComments[videoId] ?? [];
  },
};

/* ---------- いいねサービス ---------- */

// メモリ上のいいね状態（モック用）
const likedSet = new Set<string>();

export const likeService = {
  /** いいね状態を取得する */
  async getLikeStatus(videoId: string): Promise<boolean> {
    await delay(100);
    return likedSet.has(videoId);
  },

  /** いいねを切り替える */
  async toggleLike(videoId: string): Promise<{ liked: boolean; like_count: number }> {
    await delay(100);
    const video = mockVideos.find((v) => v.id === videoId);
    if (!video) throw new Error("Video not found");

    const wasLiked = likedSet.has(videoId);
    if (wasLiked) {
      likedSet.delete(videoId);
      video.like_count = Math.max(0, video.like_count - 1);
    } else {
      likedSet.add(videoId);
      video.like_count += 1;
    }

    return { liked: !wasLiked, like_count: video.like_count };
  },
};

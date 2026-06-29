/* ===================================================
 * hooks/useLike.ts
 * [保護] いいね操作を行うカスタムフック
 * バックエンドの仕様に合わせ、いいね追加は PUT、
 * いいね解除は DELETE を使い分ける。
 * =================================================== */

import { useState, useEffect, useCallback } from "react";
import { likeService } from "../services/api";

interface UseLikeResult {
  liked: boolean;
  likeCount: number;
  toggleLike: () => void;
}

export function useLike(videoId: string | undefined, initialCount: number): UseLikeResult {
  const [liked, setLiked] = useState(false);
  const [likeCount, setLikeCount] = useState(initialCount);

  // ログイン中のいいね状態をサーバーから取得する
  useEffect(() => {
    if (!videoId) return;
    likeService.getLikeStatus(videoId).then(setLiked);
  }, [videoId]);

  // 親から渡された初期カウントに追随する
  useEffect(() => {
    setLikeCount(initialCount);
  }, [initialCount]);

  /**
   * いいね状態をトグルする。
   * - 現在未いいね → PUT でいいね追加
   * - 現在いいね済み → DELETE でいいね解除
   */
  const toggleLike = useCallback(() => {
    if (!videoId) return;

    const action = liked
      ? likeService.removeLike(videoId)
      : likeService.addLike(videoId);

    action.then((result) => {
      setLiked(result.liked);
      setLikeCount(result.like_count);
    }).catch(() => {
      // 失敗時は状態を変えない（楽観的更新なし）
    });
  }, [videoId, liked]);

  return { liked, likeCount, toggleLike };
}

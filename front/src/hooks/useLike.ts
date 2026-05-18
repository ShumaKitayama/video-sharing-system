/* ===================================================
 * hooks/useLike.ts
 * [保護] いいね操作を行うカスタムフック
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

  useEffect(() => {
    if (!videoId) return;
    likeService.getLikeStatus(videoId).then(setLiked);
  }, [videoId]);

  useEffect(() => {
    setLikeCount(initialCount);
  }, [initialCount]);

  const toggleLike = useCallback(() => {
    if (!videoId) return;
    likeService.toggleLike(videoId).then((result) => {
      setLiked(result.liked);
      setLikeCount(result.like_count);
    });
  }, [videoId]);

  return { liked, likeCount, toggleLike };
}

/* ===================================================
 * hooks/useComments.ts
 * [保護] コメント一覧を取得するカスタムフック
 * =================================================== */

import { useState, useEffect } from "react";
import type { Comment } from "../types";
import { commentService } from "../services/api";

interface UseCommentsResult {
  comments: Comment[];
  loading: boolean;
}

export function useComments(videoId: string | undefined): UseCommentsResult {
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!videoId) return;

    let cancelled = false;
    setLoading(true);

    commentService
      .getComments(videoId)
      .then((data) => {
        if (!cancelled) setComments(data);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [videoId]);

  return { comments, loading };
}

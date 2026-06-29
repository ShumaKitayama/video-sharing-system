/* ===================================================
 * hooks/useComments.ts
 * [保護] コメント一覧の取得・投稿を管理するカスタムフック
 * =================================================== */

import { useState, useEffect, useCallback } from "react";
import type { Comment } from "../types";
import { commentService } from "../services/api";

interface UseCommentsResult {
  comments: Comment[];
  loading: boolean;
  posting: boolean;
  postComment: (body: string) => Promise<boolean>;
  deleteComment: (commentId: string) => Promise<void>;
}

export function useComments(videoId: string | undefined): UseCommentsResult {
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState(true);
  const [posting, setPosting] = useState(false);

  // コメント一覧を取得する
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

  /** コメントを投稿する。成功したらリストの先頭に追加する */
  const postComment = useCallback(
    async (body: string): Promise<boolean> => {
      if (!videoId || !body.trim()) return false;
      setPosting(true);
      try {
        const newComment = await commentService.postComment(videoId, body.trim());
        setComments((prev) => [newComment, ...prev]);
        return true;
      } catch {
        return false;
      } finally {
        setPosting(false);
      }
    },
    [videoId],
  );

  /** コメントを削除する */
  const deleteComment = useCallback(
    async (commentId: string): Promise<void> => {
      await commentService.deleteComment(commentId);
      setComments((prev) => prev.filter((c) => c.id !== commentId));
    },
    [],
  );

  return { comments, loading, posting, postComment, deleteComment };
}

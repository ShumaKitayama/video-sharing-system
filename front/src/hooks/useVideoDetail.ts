/* ===================================================
 * hooks/useVideoDetail.ts
 * [保護] 動画詳細を取得するカスタムフック
 * =================================================== */

import { useState, useEffect } from "react";
import type { Video } from "../types";
import { videoService } from "../services/api";

interface UseVideoDetailResult {
  video: Video | null;
  loading: boolean;
  error: string | null;
}

export function useVideoDetail(id: string | undefined): UseVideoDetailResult {
  const [video, setVideo] = useState<Video | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;

    let cancelled = false;
    setLoading(true);
    setError(null);

    videoService
      .getVideoById(id)
      .then((data) => {
        if (!cancelled) setVideo(data);
      })
      .catch((e) => {
        if (!cancelled)
          setError(e instanceof Error ? e.message : "動画の取得に失敗しました");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [id]);

  return { video, loading, error };
}

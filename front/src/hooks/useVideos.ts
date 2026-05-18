/* ===================================================
 * hooks/useVideos.ts
 * [保護] 動画一覧を取得するカスタムフック
 * =================================================== */

import { useState, useEffect, useCallback } from "react";
import type { Video, VideoQueryParams } from "../types";
import { videoService } from "../services/api";

interface UseVideosResult {
  videos: Video[];
  loading: boolean;
  error: string | null;
  totalPages: number;
  refetch: () => void;
}

export function useVideos(params?: VideoQueryParams): UseVideosResult {
  const [videos, setVideos] = useState<Video[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [totalPages, setTotalPages] = useState(0);

  const paramsKey = JSON.stringify(params);

  const fetchVideos = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await videoService.getVideos(params);
      setVideos(result.data);
      setTotalPages(result.meta.total_pages);
    } catch (e) {
      setError(e instanceof Error ? e.message : "動画の取得に失敗しました");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paramsKey]);

  useEffect(() => {
    fetchVideos();
  }, [fetchVideos]);

  return { videos, loading, error, totalPages, refetch: fetchVideos };
}

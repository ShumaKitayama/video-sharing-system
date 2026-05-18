/* ===================================================
 * hooks/useVideoUpload.ts
 * [保護] 動画アップロードを抽象化するカスタムフック
 * =================================================== */

import { useState, useCallback } from "react";

interface UseVideoUploadResult {
  uploading: boolean;
  progress: number;
  error: string | null;
  uploadVideo: (file: File, title: string, description: string) => Promise<boolean>;
  reset: () => void;
}

export function useVideoUpload(): UseVideoUploadResult {
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const uploadVideo = useCallback(
    async (file: File, title: string, description: string): Promise<boolean> => {
      // バリデーション
      const allowedTypes = ["video/mp4", "video/webm"];
      if (!allowedTypes.includes(file.type)) {
        setError("MP4またはWebM形式の動画ファイルを選択してください");
        return false;
      }
      if (file.size > 500 * 1024 * 1024) {
        setError("ファイルサイズは500MB以下にしてください");
        return false;
      }
      if (!title.trim() || title.length > 80) {
        setError("タイトルは1〜80文字で入力してください");
        return false;
      }
      if (description.length > 1000) {
        setError("説明は1000文字以下で入力してください");
        return false;
      }

      setUploading(true);
      setError(null);
      setProgress(0);

      // モック: プログレスバーのアニメーション
      for (let i = 0; i <= 100; i += 10) {
        await new Promise((r) => setTimeout(r, 150));
        setProgress(i);
      }

      setUploading(false);
      setProgress(100);

      // モックなので常に成功。将来はここで実際のAPI呼び出しを行う
      void file;
      void description;
      return true;
    },
    []
  );

  const reset = useCallback(() => {
    setUploading(false);
    setProgress(0);
    setError(null);
  }, []);

  return { uploading, progress, error, uploadVideo, reset };
}

/* ===================================================
 * hooks/useVideoUpload.ts
 * [保護] 動画アップロードを抽象化するカスタムフック
 * multipart/form-data で POST /api/v1/videos を呼ぶ。
 * Content-Type はブラウザに自動設定させるため手動設定しない。
 * =================================================== */

import { useState, useCallback } from "react";
import { API_BASE_URL, endpoints } from "../api/endpoints";

/** ローカルファイルから再生時間（秒）を読み取る */
function readVideoDurationSeconds(file: File): Promise<number | null> {
  return new Promise((resolve) => {
    const url = URL.createObjectURL(file);
    const video = document.createElement("video");
    video.preload = "metadata";

    const cleanup = () => {
      URL.revokeObjectURL(url);
      video.removeAttribute("src");
      video.load();
    };

    video.onloadedmetadata = () => {
      const seconds = Math.floor(video.duration);
      cleanup();
      resolve(Number.isFinite(seconds) && seconds >= 0 ? seconds : null);
    };

    video.onerror = () => {
      cleanup();
      resolve(null);
    };

    video.src = url;
  });
}

interface UploadedVideo {
  id: string;
  title: string;
  status: string;
}

interface UseVideoUploadResult {
  uploading: boolean;
  progress: number;
  error: string | null;
  uploadVideo: (
    file: File,
    title: string,
    description: string,
  ) => Promise<UploadedVideo | null>;
  reset: () => void;
}

export function useVideoUpload(): UseVideoUploadResult {
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const uploadVideo = useCallback(
    async (
      file: File,
      title: string,
      description: string,
    ): Promise<UploadedVideo | null> => {
      // クライアント側バリデーション
      const allowedTypes = ["video/mp4", "video/webm"];
      if (!allowedTypes.includes(file.type)) {
        setError("MP4またはWebM形式の動画ファイルを選択してください");
        return null;
      }
      if (file.size > 500 * 1024 * 1024) {
        setError("ファイルサイズは500MB以下にしてください");
        return null;
      }
      if (!title.trim() || title.length > 80) {
        setError("タイトルは1〜80文字で入力してください");
        return null;
      }
      if (description.length > 1000) {
        setError("説明は1000文字以下で入力してください");
        return null;
      }

      setUploading(true);
      setError(null);
      setProgress(0);

      const durationSeconds = await readVideoDurationSeconds(file);

      // XMLHttpRequest でアップロード進捗を取得する
      // fetch ではアップロード進捗が取れないため XHR を使用
      return new Promise<UploadedVideo | null>((resolve) => {
        const form = new FormData();
        form.append("file", file);
        form.append("title", title.trim());
        form.append("description", description);
        if (durationSeconds != null) {
          form.append("duration_seconds", String(durationSeconds));
        }

        const xhr = new XMLHttpRequest();

        xhr.upload.addEventListener("progress", (e) => {
          if (e.lengthComputable) {
            setProgress(Math.round((e.loaded / e.total) * 100));
          }
        });

        xhr.addEventListener("load", () => {
          setUploading(false);
          setProgress(100);
          if (xhr.status === 201) {
            try {
              const body = JSON.parse(xhr.responseText);
              resolve(body.data as UploadedVideo);
            } catch {
              setError("レスポンスの解析に失敗しました");
              resolve(null);
            }
          } else {
            try {
              const body = JSON.parse(xhr.responseText);
              const msg = body.error?.message ?? "アップロードに失敗しました";
              if (xhr.status === 413) {
                setError("ファイルが大きすぎます（上限500MB）");
              } else if (xhr.status === 415) {
                setError("MP4またはWebM形式のみアップロードできます");
              } else if (xhr.status === 429) {
                setError("アップロード頻度が高すぎます。しばらく待ってから再試行してください");
              } else {
                setError(msg);
              }
            } catch {
              setError("アップロードに失敗しました");
            }
            resolve(null);
          }
        });

        xhr.addEventListener("error", () => {
          setUploading(false);
          setError("ネットワークエラーが発生しました");
          resolve(null);
        });

        xhr.open("POST", `${API_BASE_URL}${endpoints.videos.create}`);
        xhr.withCredentials = true;
        // Content-Type は設定しない（XHRがmultipart/form-dataとboundaryを自動設定する）
        xhr.send(form);
      });
    },
    [],
  );

  const reset = useCallback(() => {
    setUploading(false);
    setProgress(0);
    setError(null);
  }, []);

  return { uploading, progress, error, uploadVideo, reset };
}

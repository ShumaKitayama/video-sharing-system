/* ===================================================
 * hooks/useVideoUpload.ts
 * [保護] 動画アップロードを抽象化するカスタムフック
 *
 * 本番(Vercel) / Docker からリモート API に接続時:
 *   リモート API には「リクエストボディ約4.5MB」の上限があり、
 *   動画本体を API へ POST すると 413 で弾かれる。そのため
 *   API から「署名付きアップロードURL」を1回だけ発行してもらい、
 *   ブラウザからクラウドストレージ(Cloudflare R2)へ直接 PUT する。
 *   完了後に小さな JSON だけを API に送って DB 登録する。
 *
 * ローカル開発（ローカルの Go API に接続時）:
 *   従来どおり multipart で API に直接送り、サーバーのディスクに保存する。
 *
 * 学生・UI 側は uploadVideo() を呼ぶだけでよい。
 * =================================================== */

import { useState, useCallback } from "react";
import {
  API_BASE_URL,
  API_DIRECT_BASE_URL,
  USE_REMOTE_BACKEND,
  endpoints,
} from "../api/endpoints";

/**
 * ストレージへの直接アップロード方式を使うか。
 * 本番に加え、Docker からリモート API に繋いでいるときも有効にする
 * （リモート API はリクエストボディ約4.5MBの上限があるため）。
 */
const USE_DIRECT_UPLOAD = import.meta.env.PROD || USE_REMOTE_BACKEND;

/** 受け付ける動画形式と上限サイズ（サーバー側の制限と揃えている） */
const ALLOWED_TYPES = ["video/mp4", "video/webm"];
const MAX_FILE_BYTES = 500 * 1024 * 1024;

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

/** 短命アップロードトークンを取得する（Cookie 認証・同一オリジン） */
async function fetchUploadToken(): Promise<string> {
  const res = await fetch(`${API_BASE_URL}${endpoints.uploads.token}`, {
    method: "POST",
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error("AUTH");
  }
  const body = await res.json();
  const token = body?.data?.token;
  if (!token) {
    throw new Error("TOKEN");
  }
  return token as string;
}

/** API が発行する1回限りのアップロード枠 */
interface DirectUploadSlot {
  /** ここへ動画ファイルを1回だけ PUT する */
  uploadUrl: string;
  /** アップロード完了後にファイルが公開されるURL */
  publicUrl: string;
  /** PUT 時に必ずこの値を Content-Type に指定する（署名に含まれるため） */
  contentType: string;
}

/** アップロード枠の発行を API に依頼する */
async function requestDirectUploadSlot(
  uploadToken: string,
  contentType: string,
): Promise<DirectUploadSlot> {
  const res = await fetch(`${API_BASE_URL}${endpoints.uploads.direct}`, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      "X-Upload-Token": uploadToken,
    },
    body: JSON.stringify({ content_type: contentType }),
  });

  const body = await res.json().catch(() => null);
  if (!res.ok) {
    const msg = body?.error?.message ?? "アップロードの準備に失敗しました";
    throw new Error(msg);
  }

  const slot = body?.data;
  if (!slot?.upload_url || !slot?.public_url) {
    throw new Error("アップロードの準備に失敗しました");
  }
  return {
    uploadUrl: slot.upload_url as string,
    publicUrl: slot.public_url as string,
    contentType: (slot.content_type as string) ?? contentType,
  };
}

/**
 * 署名付きURLへ動画ファイルを直接 PUT する（進捗付き XHR）。
 * ストレージの認証情報はブラウザに一切渡らない。
 */
function putFileToStorage(
  slot: DirectUploadSlot,
  file: File,
  onProgress: (percent: number) => void,
): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();

    xhr.upload.addEventListener("progress", (e) => {
      if (e.lengthComputable) {
        // 100% は DB 登録が終わってから出したいので 99% で止めておく
        onProgress(Math.min(99, Math.round((e.loaded / e.total) * 100)));
      }
    });

    xhr.addEventListener("load", () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
        return;
      }
      reject(new Error("動画の送信に失敗しました。通信環境を確認してください"));
    });

    xhr.addEventListener("error", () => {
      reject(new Error("ネットワークエラーが発生しました"));
    });

    xhr.addEventListener("abort", () => {
      reject(new Error("アップロードが中断されました"));
    });

    xhr.open("PUT", slot.uploadUrl);
    // 署名した値と1文字でも違うとストレージ側で拒否される
    xhr.setRequestHeader("Content-Type", slot.contentType);
    // 別オリジンへの送信なので Cookie は付けない
    xhr.withCredentials = false;
    xhr.send(file);
  });
}

/** DB に動画レコードを登録する（ファイル本体は送らない小さな JSON） */
async function registerVideo(
  uploadToken: string,
  params: {
    fileUrl: string;
    title: string;
    description: string;
    contentType: string;
    durationSeconds: number | null;
  },
): Promise<UploadedVideo> {
  const res = await fetch(`${API_BASE_URL}${endpoints.videos.create}`, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      "X-Upload-Token": uploadToken,
    },
    body: JSON.stringify({
      blob_url: params.fileUrl,
      title: params.title,
      description: params.description,
      content_type: params.contentType,
      duration_seconds:
        params.durationSeconds != null ? params.durationSeconds : undefined,
    }),
  });

  const body = await res.json().catch(() => null);
  if (!res.ok) {
    const msg = body?.error?.message ?? "アップロードに失敗しました";
    throw new Error(msg);
  }
  return body.data as UploadedVideo;
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
      if (!ALLOWED_TYPES.includes(file.type)) {
        setError("MP4またはWebM形式の動画ファイルを選択してください");
        return null;
      }
      if (file.size > MAX_FILE_BYTES) {
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

      try {
        const uploadToken = await fetchUploadToken();

        if (USE_DIRECT_UPLOAD) {
          // ブラウザ → クラウドストレージへ直接アップロード（大容量OK）
          const slot = await requestDirectUploadSlot(uploadToken, file.type);
          await putFileToStorage(slot, file, setProgress);

          const created = await registerVideo(uploadToken, {
            fileUrl: slot.publicUrl,
            title: title.trim(),
            description,
            contentType: file.type,
            durationSeconds,
          });
          setProgress(100);
          setUploading(false);
          return created;
        }

        // ローカル開発: 従来どおり multipart で API に直接送る
        const created = await uploadMultipart(uploadToken, file, {
          title: title.trim(),
          description,
          durationSeconds,
          onProgress: (p) => setProgress(p),
        });
        setProgress(100);
        setUploading(false);
        return created;
      } catch (err) {
        setUploading(false);
        const message = err instanceof Error ? err.message : "";
        if (message === "AUTH") {
          setError("ログインが必要です。再度ログインしてからお試しください");
        } else if (message === "TOKEN") {
          setError("アップロードトークンの取得に失敗しました");
        } else if (message) {
          setError(message);
        } else {
          setError("アップロードに失敗しました");
        }
        return null;
      }
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

/** ローカル開発用: multipart アップロード（進捗付き XHR） */
function uploadMultipart(
  uploadToken: string,
  file: File,
  opts: {
    title: string;
    description: string;
    durationSeconds: number | null;
    onProgress: (percent: number) => void;
  },
): Promise<UploadedVideo> {
  return new Promise<UploadedVideo>((resolve, reject) => {
    const form = new FormData();
    form.append("file", file);
    form.append("title", opts.title);
    form.append("description", opts.description);
    if (opts.durationSeconds != null) {
      form.append("duration_seconds", String(opts.durationSeconds));
    }

    const xhr = new XMLHttpRequest();

    xhr.upload.addEventListener("progress", (e) => {
      if (e.lengthComputable) {
        opts.onProgress(Math.round((e.loaded / e.total) * 100));
      }
    });

    xhr.addEventListener("load", () => {
      let body: unknown = null;
      try {
        body = JSON.parse(xhr.responseText);
      } catch {
        // レスポンスが JSON でない場合は body は null のまま
      }
      if (xhr.status === 201) {
        const data = (body as { data?: UploadedVideo } | null)?.data;
        if (data) {
          resolve(data);
        } else {
          reject(new Error("レスポンスの解析に失敗しました"));
        }
        return;
      }
      const msg =
        (body as { error?: { message?: string } } | null)?.error?.message ??
        "アップロードに失敗しました";
      reject(new Error(msg));
    });

    xhr.addEventListener("error", () => {
      reject(new Error("ネットワークエラーが発生しました"));
    });

    xhr.open("POST", `${API_DIRECT_BASE_URL}${endpoints.videos.create}`);
    xhr.setRequestHeader("X-Upload-Token", uploadToken);
    xhr.withCredentials = true;
    xhr.send(form);
  });
}

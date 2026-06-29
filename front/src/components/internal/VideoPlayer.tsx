/* ===================================================
 * components/internal/VideoPlayer.tsx
 * [保護] 動画プレーヤーラッパー
 * バックエンドの Range 対応ストリーミングエンドポイントに
 * <video> の src を直接指定する。
 * fetch で丸ごと読み込まず、ブラウザに Range Request を任せる。
 * =================================================== */

import { API_BASE_URL, endpoints } from "../../api/endpoints";

interface VideoPlayerProps {
  videoId: string;
}

export default function VideoPlayer({ videoId }: VideoPlayerProps) {
  const src = `${API_BASE_URL}${endpoints.videos.stream(videoId)}`;

  return (
    <div className="player" id="video-player">
      <video
        className="player__video"
        controls
        src={src}
        id={`video-${videoId}`}
        style={{ width: "100%", height: "100%", display: "block", background: "#000" }}
      >
        お使いのブラウザは動画再生に対応していません。
      </video>
    </div>
  );
}

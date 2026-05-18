/* ===================================================
 * components/internal/VideoPlayer.tsx
 * [保護] 動画プレーヤーラッパー
 * =================================================== */

interface VideoPlayerProps {
  videoId: string;
  thumbnailUrl?: string;
}

export default function VideoPlayer({ thumbnailUrl }: VideoPlayerProps) {
  return (
    <div className="player" id="video-player">
      <div className="player__placeholder" style={{ background: thumbnailUrl || "linear-gradient(135deg, #1a1a2e, #16213e)" }}>
        <div className="player__play-btn">▶</div>
        <span className="player__text">バックエンド接続時に動画が再生されます</span>
      </div>
    </div>
  );
}

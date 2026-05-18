/* ===================================================
 * components/student/VideoGrid.tsx
 * [学生編集可] 動画カードのグリッドレイアウト
 * =================================================== */

import type { Video } from "../../types";
import VideoCard from "./VideoCard";

interface VideoGridProps {
  videos: Video[];
  loading?: boolean;
}

function SkeletonCard() {
  return (
    <div className="video-card">
      <div className="skeleton skeleton--thumb" />
      <div className="video-card__info">
        <div className="skeleton skeleton--avatar" />
        <div className="video-card__details">
          <div className="skeleton skeleton--title" />
          <div className="skeleton skeleton--text" />
        </div>
      </div>
    </div>
  );
}

export default function VideoGrid({ videos, loading }: VideoGridProps) {
  if (loading) {
    return (
      <div className="video-grid">
        {Array.from({ length: 8 }).map((_, i) => (
          <SkeletonCard key={i} />
        ))}
      </div>
    );
  }

  if (videos.length === 0) {
    return (
      <div style={{ textAlign: "center", padding: "48px 0", color: "var(--text-secondary)" }}>
        <div style={{ fontSize: 48, marginBottom: 16 }}>📭</div>
        <p>動画が見つかりませんでした</p>
      </div>
    );
  }

  return (
    <div className="video-grid">
      {videos.map((video) => (
        <VideoCard key={video.id} video={video} />
      ))}
    </div>
  );
}

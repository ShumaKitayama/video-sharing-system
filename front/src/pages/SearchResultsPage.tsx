/* ===================================================
 * pages/SearchResultsPage.tsx
 * 検索結果ページ
 * =================================================== */

import { useSearchParams, useNavigate } from "react-router-dom";
import { useState } from "react";
import { useVideos } from "../hooks/useVideos";
import ChannelAvatar from "../components/student/ChannelAvatar";
import VideoPreview from "../components/internal/VideoPreview";
import { formatDuration, formatViewCount, formatTimeAgo } from "../components/student/VideoCard";
import type { Video } from "../types";

interface SearchResultItemProps {
  video: Video;
  onNavigate: () => void;
}

function SearchResultItem({ video, onNavigate }: SearchResultItemProps) {
  const [detectedDuration, setDetectedDuration] = useState<number | null>(null);
  const durationLabel = formatDuration(video.duration_seconds ?? detectedDuration);

  return (
    <div className="search-result" onClick={onNavigate} id={`search-result-${video.id}`}>
      <div className="search-result__thumb">
        <VideoPreview
          videoId={video.id}
          onDurationLoaded={(seconds) =>
            setDetectedDuration((prev) => prev ?? seconds)
          }
        />
        {durationLabel && (
          <span className="search-result__duration">{durationLabel}</span>
        )}
      </div>
      <div className="search-result__info">
        <h3 className="search-result__title">{video.title}</h3>
        <div className="search-result__meta">
          {formatViewCount(video.view_count)} 回視聴 • {formatTimeAgo(video.created_at)}
        </div>
        <div className="search-result__channel">
          <ChannelAvatar displayName={video.uploader.display_name} size={24} />
          {video.uploader.display_name}
        </div>
        <div className="search-result__desc">{video.description}</div>
      </div>
    </div>
  );
}

export default function SearchResultsPage() {
  const [searchParams] = useSearchParams();
  const q = searchParams.get("q") ?? "";
  const navigate = useNavigate();
  const { videos, loading } = useVideos({ q });

  return (
    <div className="search-results fade-in" id="search-page">
      <h2 style={{ fontSize: 16, color: "var(--text-secondary)", marginBottom: 20 }}>
        「{q}」の検索結果 ({videos.length}件)
      </h2>

      {loading ? (
        <div style={{ color: "var(--text-secondary)" }}>検索中...</div>
      ) : videos.length === 0 ? (
        <div style={{ textAlign: "center", padding: 48, color: "var(--text-secondary)" }}>
          <div style={{ fontSize: 48, marginBottom: 16 }}>🔍</div>
          <p>検索結果が見つかりませんでした</p>
        </div>
      ) : (
        videos.map((v) => (
          <SearchResultItem key={v.id} video={v} onNavigate={() => navigate(`/watch/${v.id}`)} />
        ))
      )}
    </div>
  );
}

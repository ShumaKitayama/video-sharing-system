/* ===================================================
 * pages/SearchResultsPage.tsx
 * 検索結果ページ
 * =================================================== */

import { useSearchParams, useNavigate } from "react-router-dom";
import { useVideos } from "../hooks/useVideos";
import ChannelAvatar from "../components/student/ChannelAvatar";
import { formatDuration, formatViewCount, formatTimeAgo } from "../components/student/VideoCard";

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
          <div key={v.id} className="search-result" onClick={() => navigate(`/watch/${v.id}`)} id={`search-result-${v.id}`}>
            <div className="search-result__thumb" style={{ background: v.thumbnail_url || "var(--bg-elevated)" }}>
              🎬
              <span className="search-result__duration">{formatDuration(v.duration_seconds)}</span>
            </div>
            <div className="search-result__info">
              <h3 className="search-result__title">{v.title}</h3>
              <div className="search-result__meta">
                {formatViewCount(v.view_count)} 回視聴 • {formatTimeAgo(v.created_at)}
              </div>
              <div className="search-result__channel">
                <ChannelAvatar displayName={v.uploader.display_name} size={24} />
                {v.uploader.display_name}
              </div>
              <div className="search-result__desc">{v.description}</div>
            </div>
          </div>
        ))
      )}
    </div>
  );
}

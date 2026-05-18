/* ===================================================
 * pages/WatchPage.tsx
 * 動画再生ページ
 * =================================================== */

import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useVideoDetail } from "../hooks/useVideoDetail";
import { useComments } from "../hooks/useComments";
import { useLike } from "../hooks/useLike";
import { useVideos } from "../hooks/useVideos";
import VideoPlayer from "../components/internal/VideoPlayer";
import CommentList from "../components/student/CommentList";
import ChannelAvatar from "../components/student/ChannelAvatar";
import { formatDuration, formatViewCount, formatTimeAgo } from "../components/student/VideoCard";

export default function WatchPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { video, loading, error } = useVideoDetail(id);
  const { comments, loading: commentsLoading } = useComments(id);
  const { liked, likeCount, toggleLike } = useLike(id, video?.like_count ?? 0);
  const { videos: relatedVideos } = useVideos({ per_page: 10 });
  const [descExpanded, setDescExpanded] = useState(false);

  if (loading) return <div style={{ padding: 48, textAlign: "center", color: "var(--text-secondary)" }}>読み込み中...</div>;
  if (error || !video) return <div style={{ padding: 48, textAlign: "center", color: "var(--text-secondary)" }}>動画が見つかりません</div>;

  const related = relatedVideos.filter((v) => v.id !== id);

  return (
    <div className="watch fade-in" id="watch-page">
      <div className="watch__main">
        <VideoPlayer videoId={video.id} thumbnailUrl={video.thumbnail_url} />

        <h1 className="video-info__title">{video.title}</h1>

        <div className="video-info__actions">
          <div className="video-info__left">
            <ChannelAvatar displayName={video.uploader.display_name} size={40} />
            <div>
              <div className="video-info__channel-name" style={{ cursor: "pointer" }} onClick={() => navigate(`/channel/${video.uploader.id}`)}>
                {video.uploader.display_name}
              </div>
            </div>
          </div>
          <div className="video-info__right">
            <button className={`video-info__btn ${liked ? "video-info__btn--liked" : ""}`} onClick={toggleLike} id="like-btn">
              {liked ? "👍" : "👍"} {formatViewCount(likeCount)}
            </button>
            <button className="video-info__btn">🔗 共有</button>
          </div>
        </div>

        <div className="video-desc" onClick={() => setDescExpanded(!descExpanded)}>
          <div className="video-desc__meta">
            {formatViewCount(video.view_count)} 回視聴 • {formatTimeAgo(video.created_at)}
          </div>
          <div className={`video-desc__text ${!descExpanded ? "video-desc__text--collapsed" : ""}`}>
            {video.description}
          </div>
          <div className="video-desc__toggle">{descExpanded ? "一部を表示" : "もっと見る"}</div>
        </div>

        <CommentList comments={comments} loading={commentsLoading} />
      </div>

      <aside className="watch__sidebar">
        {related.map((v) => (
          <div key={v.id} className="related-video" onClick={() => navigate(`/watch/${v.id}`)} id={`related-${v.id}`}>
            <div className="related-video__thumb" style={{ background: v.thumbnail_url || "var(--bg-elevated)" }}>
              🎬
              <span className="related-video__duration">{formatDuration(v.duration_seconds)}</span>
            </div>
            <div className="related-video__info">
              <div className="related-video__title">{v.title}</div>
              <div className="related-video__channel">{v.uploader.display_name}</div>
              <div className="related-video__meta">{formatViewCount(v.view_count)} 回視聴 • {formatTimeAgo(v.created_at)}</div>
            </div>
          </div>
        ))}
      </aside>
    </div>
  );
}

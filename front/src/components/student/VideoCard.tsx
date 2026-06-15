/* ===================================================
 * components/student/VideoCard.tsx
 * [学生編集可] 動画カード — ソーシャルフィードスタイル
 * アバター+ユーザー名がサムネイル上にオーバーレイ、
 * 右側にシェア/ハートボタン配置
 * =================================================== */

import { useNavigate } from "react-router-dom";
import type { Video } from "../../types";
import ChannelAvatar from "./ChannelAvatar";

interface VideoCardProps {
  video: Video;
}

function formatDuration(seconds: number | null): string {
  if (seconds === null) return "0:00";
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

function formatViewCount(count: number): string {
  if (count >= 10000) return `${(count / 10000).toFixed(1)}万`;
  if (count >= 1000) return `${(count / 1000).toFixed(1)}千`;
  return `${count}`;
}

function formatTimeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 60) return `${minutes}分前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}時間前`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}日前`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}ヶ月前`;
  return `${Math.floor(months / 12)}年前`;
}

export default function VideoCard({ video }: VideoCardProps) {
  const navigate = useNavigate();

  return (
    <div className="video-card fade-in" onClick={() => navigate(`/watch/${video.id}`)} id={`video-card-${video.id}`}>
      <div className="video-card__thumbnail">
        <div className="video-card__thumbnail-img" style={{ background: video.thumbnail_url || "var(--bg-hover)" }}>
          🎬
        </div>

        {/* オーバーレイ: ユーザー名 + 再生時間 + アクション */}
        <div className="video-card__overlay">
          <div className="video-card__overlay-top" onClick={(e) => { e.stopPropagation(); navigate(`/channel/${video.uploader.id}`); }}>
            <ChannelAvatar displayName={video.uploader.display_name} size={32} />
            <span className="video-card__overlay-username">{video.uploader.display_name}</span>
          </div>
          <div className="video-card__overlay-bottom">
            <span className="video-card__duration">{formatDuration(video.duration_seconds)}</span>
          </div>
        </div>

        {/* 右サイドアクションボタン */}
        <div className="video-card__actions">
          <button className="video-card__action-btn" onClick={(e) => e.stopPropagation()} title="いいね">
            ♡
          </button>
        </div>
      </div>

      <div className="video-card__info">
        <h3 className="video-card__title">{video.title}</h3>
        <div className="video-card__meta">
          <span>{formatViewCount(video.view_count)} 回視聴</span>
          <span className="video-card__meta-dot" />
          <span>{formatTimeAgo(video.created_at)}</span>
        </div>
      </div>
    </div>
  );
}

export { formatDuration, formatViewCount, formatTimeAgo };

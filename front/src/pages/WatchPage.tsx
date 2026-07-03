/* ===================================================
 * pages/WatchPage.tsx
 * 動画再生ページ — 全画面風イマーシブスタイル
 * 動画を大きく表示し、タイトル・説明・アクションを
 * 画面内にオーバーレイする
 * =================================================== */

import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useVideoDetail } from "../hooks/useVideoDetail";
import { useComments } from "../hooks/useComments";
import { useLike } from "../hooks/useLike";
import { useAuth } from "../hooks/useAuth";
import { useVideoDelete } from "../hooks/useVideoDelete";
import VideoPlayer from "../components/internal/VideoPlayer";
import ChannelAvatar from "../components/student/ChannelAvatar";
import CommentItem from "../components/student/CommentItem";
import { formatViewCount, formatTimeAgo } from "../components/student/VideoCard";

export default function WatchPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { video, loading, error } = useVideoDetail(id);
  const { comments, loading: commentsLoading, posting, postComment } = useComments(id);
  const { liked, likeCount, toggleLike } = useLike(id, video?.like_count ?? 0);
  const { user } = useAuth();
  const { deleteVideo, deleting } = useVideoDelete();
  const [showComments, setShowComments] = useState(false);
  const [descExpanded, setDescExpanded] = useState(false);
  const [commentText, setCommentText] = useState("");

  if (loading) return <div style={{ padding: 48, textAlign: "center", color: "var(--text-secondary)" }}>読み込み中...</div>;
  if (error || !video) return <div style={{ padding: 48, textAlign: "center", color: "var(--text-secondary)" }}>動画が見つかりません</div>;

  const canDelete =
    Boolean(user) &&
    (user!.id === video.uploader.id || user!.role === "teacher");
  const isOwnUploaderChannel = user?.id === video.uploader.id;

  const handleDelete = async () => {
    const ok = await deleteVideo(video.id);
    if (ok) navigate(isOwnUploaderChannel ? `/channel/${video.uploader.id}` : "/");
  };

  const handlePostComment = async () => {
    if (!commentText.trim()) return;
    const ok = await postComment(commentText);
    if (ok) setCommentText("");
  };

  return (
    <div className="watch-immersive fade-in" id="watch-page">
      {/* メインプレーヤーエリア */}
      <div className="watch-immersive__player">
        {/* 動画プレーヤー */}
        <VideoPlayer videoId={video.id} variant="immersive" />

        {/* 左上：戻るボタン */}
        <button className="watch-immersive__back" onClick={() => navigate(-1)} id="back-btn">
          ← 戻る
        </button>

        {/* 右サイドアクション */}
        <div className="watch-immersive__side-actions">
          <button
            className={`watch-immersive__action-btn ${liked ? "watch-immersive__action-btn--active" : ""}`}
            onClick={toggleLike}
            id="like-btn"
          >
            <span className="watch-immersive__action-icon">{liked ? "❤️" : "♡"}</span>
            <span className="watch-immersive__action-label">{formatViewCount(likeCount)}</span>
          </button>
          <button
            className="watch-immersive__action-btn"
            onClick={() => setShowComments(!showComments)}
            id="comment-toggle-btn"
          >
            <span className="watch-immersive__action-icon">💬</span>
            <span className="watch-immersive__action-label">{video.comment_count}</span>
          </button>
          <button className="watch-immersive__action-btn">
            <span className="watch-immersive__action-icon">👁</span>
            <span className="watch-immersive__action-label">{formatViewCount(video.view_count)}</span>
          </button>
          {canDelete && (
            <button
              className="watch-immersive__action-btn watch-immersive__action-btn--danger"
              onClick={() => void handleDelete()}
              disabled={deleting}
              id="delete-video-btn"
            >
              <span className="watch-immersive__action-icon">🗑</span>
              <span className="watch-immersive__action-label">{deleting ? "..." : "削除"}</span>
            </button>
          )}
        </div>

        {/* 下部オーバーレイ：ユーザー情報 + タイトル + 説明 */}
        <div className="watch-immersive__bottom">
          <div className="watch-immersive__user" onClick={() => navigate(`/channel/${video.uploader.id}`)}>
            <ChannelAvatar displayName={video.uploader.display_name} size={36} />
            <span className="watch-immersive__username">{video.uploader.display_name}</span>
            <span className="watch-immersive__time">{formatTimeAgo(video.created_at)}</span>
          </div>
          <h1 className="watch-immersive__title">{video.title}</h1>
          <div
            className={`watch-immersive__desc ${!descExpanded ? "watch-immersive__desc--collapsed" : ""}`}
            onClick={() => setDescExpanded(!descExpanded)}
          >
            {video.description}
          </div>
          {video.description.length > 60 && (
            <button className="watch-immersive__desc-toggle" onClick={() => setDescExpanded(!descExpanded)}>
              {descExpanded ? "閉じる" : "もっと見る"}
            </button>
          )}
        </div>
      </div>

      {/* コメントパネル（スライドイン） */}
      {showComments && (
        <div className="watch-immersive__comments-overlay" onClick={() => setShowComments(false)}>
          <div className="watch-immersive__comments-panel" onClick={(e) => e.stopPropagation()}>
            <div className="watch-immersive__comments-header">
              <h2 className="watch-immersive__comments-title">コメント ({comments.length})</h2>
              <button className="watch-immersive__comments-close" onClick={() => setShowComments(false)}>✕</button>
            </div>
            <div className="watch-immersive__comments-body">
              <div className="watch-immersive__comment-input-wrap">
                <input
                  className="watch-immersive__comment-input"
                  type="text"
                  placeholder="コメントを追加..."
                  id="comment-input"
                  value={commentText}
                  onChange={(e) => setCommentText(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handlePostComment()}
                  disabled={posting}
                />
                <button
                  onClick={handlePostComment}
                  disabled={posting || !commentText.trim()}
                  style={{
                    marginTop: 8,
                    padding: "8px 16px",
                    background: "var(--accent)",
                    color: "#fff",
                    border: "none",
                    borderRadius: 8,
                    cursor: "pointer",
                    fontSize: 13,
                    width: "100%",
                  }}
                  id="comment-submit"
                >
                  {posting ? "送信中..." : "コメントを送信"}
                </button>
              </div>
              {commentsLoading ? (
                <div style={{ padding: 24, textAlign: "center", color: "var(--text-secondary)" }}>読み込み中...</div>
              ) : comments.length === 0 ? (
                <div style={{ padding: 24, textAlign: "center", color: "var(--text-secondary)" }}>
                  まだコメントはありません
                </div>
              ) : (
                comments.map((c) => <CommentItem key={c.id} comment={c} />)
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

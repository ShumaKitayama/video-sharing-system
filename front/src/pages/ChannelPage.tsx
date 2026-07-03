/* ===================================================
 * pages/ChannelPage.tsx
 * チャンネル（ユーザー）ページ
 * =================================================== */

import { useParams } from "react-router-dom";
import { useVideos } from "../hooks/useVideos";
import { useAuth } from "../hooks/useAuth";
import { useVideoDelete } from "../hooks/useVideoDelete";
import VideoGrid from "../components/student/VideoGrid";
import ChannelAvatar from "../components/student/ChannelAvatar";

export default function ChannelPage() {
  const { id } = useParams<{ id: string }>();
  const { videos, loading, refetch } = useVideos({ uploader_id: id });
  const { user } = useAuth();
  const { deleteVideo, deleting } = useVideoDelete();
  const isOwnChannel = Boolean(user && id && user.id === id);

  const handleDelete = async (videoId: string) => {
    const ok = await deleteVideo(videoId);
    if (ok) refetch();
  };

  // 動画一覧の先頭からユーザー表示名を取得する
  const uploader = videos[0]?.uploader ?? null;

  return (
    <div className="fade-in" id="channel-page">
      <div className="channel-header">
        <ChannelAvatar displayName={uploader?.display_name ?? "?"} size={80} />
        <div>
          <h1 className="channel-header__name">
            {loading ? "読み込み中..." : (uploader?.display_name ?? "ユーザーが見つかりません")}
          </h1>
          <div className="channel-header__stats">
            {!loading && uploader && `動画 ${videos.length}本`}
          </div>
        </div>
      </div>
      <VideoGrid
        videos={videos}
        loading={loading}
        canDelete={isOwnChannel}
        deleting={deleting}
        onDelete={handleDelete}
      />
    </div>
  );
}

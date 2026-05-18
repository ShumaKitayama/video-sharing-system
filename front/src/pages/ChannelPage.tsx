/* ===================================================
 * pages/ChannelPage.tsx
 * チャンネル（ユーザー）ページ
 * =================================================== */

import { useParams } from "react-router-dom";
import { useVideos } from "../hooks/useVideos";
import { mockUsers } from "../mock/data";
import VideoGrid from "../components/student/VideoGrid";
import ChannelAvatar from "../components/student/ChannelAvatar";

export default function ChannelPage() {
  const { id } = useParams<{ id: string }>();
  const user = mockUsers.find((u) => u.id === id);
  const { videos, loading } = useVideos({ uploader_id: id });

  if (!user) {
    return (
      <div style={{ textAlign: "center", padding: 48, color: "var(--text-secondary)" }}>
        ユーザーが見つかりません
      </div>
    );
  }

  return (
    <div className="fade-in" id="channel-page">
      <div className="channel-header">
        <ChannelAvatar displayName={user.display_name} size={80} />
        <div>
          <h1 className="channel-header__name">{user.display_name}</h1>
          <div className="channel-header__stats">@{user.username} • 動画 {videos.length}本</div>
        </div>
      </div>
      <VideoGrid videos={videos} loading={loading} />
    </div>
  );
}

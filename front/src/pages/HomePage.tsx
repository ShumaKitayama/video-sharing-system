/* ===================================================
 * pages/HomePage.tsx
 * トップページ - 動画一覧グリッド
 * =================================================== */

import { useVideos } from "../hooks/useVideos";
import VideoGrid from "../components/student/VideoGrid";

export default function HomePage() {
  const { videos, loading } = useVideos({ sort: "latest" });

  return (
    <div id="home-page">
      <VideoGrid videos={videos} loading={loading} />
    </div>
  );
}

/* ===================================================
 * pages/HomePage.tsx
 * トップページ - 動画一覧グリッド
 * =================================================== */

import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useVideos } from "../hooks/useVideos";
import VideoGrid from "../components/student/VideoGrid";
import type { VideoQueryParams } from "../types";

const sortOptions = [
  { label: "すべて", value: undefined },
  { label: "最新", value: "latest" as const },
  { label: "人気", value: "most_viewed" as const },
  { label: "高評価", value: "most_liked" as const },
];

export default function HomePage() {
  const [searchParams] = useSearchParams();
  const sortFromUrl = searchParams.get("sort") as VideoQueryParams["sort"] | null;
  const [activeSort, setActiveSort] = useState<VideoQueryParams["sort"] | undefined>(sortFromUrl ?? undefined);

  const { videos, loading } = useVideos({ sort: activeSort });

  return (
    <div id="home-page">
      <div className="filter-chips">
        {sortOptions.map((opt) => (
          <button
            key={opt.label}
            className={`filter-chip ${activeSort === opt.value ? "filter-chip--active" : ""}`}
            onClick={() => setActiveSort(opt.value)}
          >
            {opt.label}
          </button>
        ))}
      </div>
      <VideoGrid videos={videos} loading={loading} />
    </div>
  );
}

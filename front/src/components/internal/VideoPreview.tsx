/* ===================================================
 * components/internal/VideoPreview.tsx
 * [保護] サムネイル用プレビュー
 * 動画ストリームの冒頭3秒をループ再生する（ミュート）。
 * 画面内に入ったときだけ再生し、LAN負荷を抑える。
 * =================================================== */

import { useEffect, useRef } from "react";
import { API_BASE_URL, endpoints } from "../../api/endpoints";

const PREVIEW_LOOP_SECONDS = 3;

interface VideoPreviewProps {
  videoId: string;
  /** メタデータ読み込み後に再生時間（秒）を親へ通知する */
  onDurationLoaded?: (seconds: number) => void;
}

export default function VideoPreview({
  videoId,
  onDurationLoaded,
}: VideoPreviewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const videoRef = useRef<HTMLVideoElement>(null);
  const durationReportedRef = useRef(false);

  useEffect(() => {
    durationReportedRef.current = false;
  }, [videoId]);

  useEffect(() => {
    const container = containerRef.current;
    const video = videoRef.current;
    if (!container || !video) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          video.play().catch(() => {});
        } else {
          video.pause();
          video.currentTime = 0;
        }
      },
      { threshold: 0.2 },
    );

    observer.observe(container);
    return () => observer.disconnect();
  }, [videoId]);

  const reportDuration = () => {
    if (durationReportedRef.current || !onDurationLoaded) return;
    const video = videoRef.current;
    if (!video || !Number.isFinite(video.duration) || video.duration <= 0) return;
    durationReportedRef.current = true;
    onDurationLoaded(Math.floor(video.duration));
  };

  const handleTimeUpdate = () => {
    const video = videoRef.current;
    if (video && video.currentTime >= PREVIEW_LOOP_SECONDS) {
      video.currentTime = 0;
    }
  };

  const src = `${API_BASE_URL}${endpoints.videos.stream(videoId)}`;

  return (
    <div className="video-preview" ref={containerRef}>
      <video
        ref={videoRef}
        className="video-preview__video"
        src={src}
        muted
        playsInline
        preload="metadata"
        onLoadedMetadata={reportDuration}
        onDurationChange={reportDuration}
        onTimeUpdate={handleTimeUpdate}
        aria-hidden
      />
    </div>
  );
}

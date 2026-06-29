/* ===================================================
 * components/internal/VideoPlayer.tsx
 * [保護] 動画プレーヤー
 * Range 対応ストリームを <video> に直接指定し、
 * イマーシブモードではカスタムコントロールを表示する。
 * =================================================== */

import { useCallback, useEffect, useRef, useState } from "react";
import { API_BASE_URL, endpoints } from "../../api/endpoints";

interface VideoPlayerProps {
  videoId: string;
  variant?: "default" | "immersive";
}

const SKIP_SECONDS = 15;

function formatVideoTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s.toString().padStart(2, "0")}`;
}

export default function VideoPlayer({
  videoId,
  variant = "default",
}: VideoPlayerProps) {
  const src = `${API_BASE_URL}${endpoints.videos.stream(videoId)}`;
  const videoRef = useRef<HTMLVideoElement>(null);

  const [playing, setPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [seeking, setSeeking] = useState(false);
  const [seekValue, setSeekValue] = useState(0);

  const isImmersive = variant === "immersive";

  const syncTime = useCallback(() => {
    const video = videoRef.current;
    if (!video || seeking) return;
    setCurrentTime(video.currentTime);
    setSeekValue(video.currentTime);
  }, [seeking]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    setPlaying(!video.paused);
    setDuration(video.duration || 0);
    setCurrentTime(video.currentTime);
    setSeekValue(video.currentTime);

    const onPlay = () => setPlaying(true);
    const onPause = () => setPlaying(false);
    const onLoadedMetadata = () => setDuration(video.duration || 0);
    const onDurationChange = () => setDuration(video.duration || 0);

    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("timeupdate", syncTime);
    video.addEventListener("loadedmetadata", onLoadedMetadata);
    video.addEventListener("durationchange", onDurationChange);

    return () => {
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("timeupdate", syncTime);
      video.removeEventListener("loadedmetadata", onLoadedMetadata);
      video.removeEventListener("durationchange", onDurationChange);
    };
  }, [syncTime, videoId]);

  const togglePlay = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;
    if (video.paused) {
      video.play().catch(() => {});
    } else {
      video.pause();
    }
  }, []);

  const skip = useCallback((delta: number) => {
    const video = videoRef.current;
    if (!video) return;
    const max = Number.isFinite(video.duration) ? video.duration : 0;
    const next = Math.max(0, Math.min(max, video.currentTime + delta));
    video.currentTime = next;
    setCurrentTime(next);
    setSeekValue(next);
  }, []);

  const handleSeekChange = (value: number) => {
    setSeekValue(value);
  };

  const handleSeekEnd = (value: number) => {
    const video = videoRef.current;
    if (video) video.currentTime = value;
    setCurrentTime(value);
    setSeekValue(value);
    setSeeking(false);
  };

  if (!isImmersive) {
    return (
      <div className="player" id="video-player">
        <video
          className="player__video"
          src={src}
          controls
          id={`video-${videoId}`}
          playsInline
          preload="metadata"
        >
          お使いのブラウザは動画再生に対応していません。
        </video>
      </div>
    );
  }

  return (
    <div className="player player--immersive" id="video-player">
      <div className="player__media">
        <video
          ref={videoRef}
          className="player__video"
          src={src}
          id={`video-${videoId}`}
          playsInline
          preload="metadata"
          onClick={togglePlay}
        >
          お使いのブラウザは動画再生に対応していません。
        </video>

        {!playing && (
          <button
            type="button"
            className="player__center-play"
            onClick={togglePlay}
            aria-label="再生"
          >
            ▶
          </button>
        )}
      </div>

      <div className="player__controls" onClick={(e) => e.stopPropagation()}>
        <input
          className="player__seek"
          type="range"
          min={0}
          max={duration || 0}
          step={0.1}
          value={seeking ? seekValue : currentTime}
          onMouseDown={() => setSeeking(true)}
          onTouchStart={() => setSeeking(true)}
          onChange={(e) => handleSeekChange(Number(e.target.value))}
          onMouseUp={(e) => handleSeekEnd(Number(e.currentTarget.value))}
          onTouchEnd={(e) => handleSeekEnd(Number(e.currentTarget.value))}
          aria-label="再生位置"
        />

        <div className="player__controls-row">
          <div className="player__controls-left">
            <button
              type="button"
              className="player__control-btn"
              onClick={() => skip(-SKIP_SECONDS)}
              aria-label="15秒戻る"
            >
              ↺ 15
            </button>
            <button
              type="button"
              className="player__control-btn player__control-btn--primary"
              onClick={togglePlay}
              aria-label={playing ? "一時停止" : "再生"}
            >
              {playing ? "⏸" : "▶"}
            </button>
            <button
              type="button"
              className="player__control-btn"
              onClick={() => skip(SKIP_SECONDS)}
              aria-label="15秒進む"
            >
              15 ↻
            </button>
          </div>

          <span className="player__time">
            {formatVideoTime(currentTime)} / {formatVideoTime(duration)}
          </span>
        </div>
      </div>
    </div>
  );
}

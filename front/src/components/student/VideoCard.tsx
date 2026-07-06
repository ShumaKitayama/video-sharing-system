/* ===================================================
 * components/student/VideoCard.tsx
 * [学生編集可] 動画カード — ソーシャルフィードスタイル
 * アバター+ユーザー名がサムネイル上にオーバーレイ、
 * 右側にシェア/ハートボタン配置
 * =================================================== */

import { useNavigate } from "react-router-dom";
import { useState } from "react";
import type { Video } from "../../types";
import ChannelAvatar from "./ChannelAvatar";
import VideoPreview from "../internal/VideoPreview";

interface VideoCardProps {
  video: Video;
  canDelete?: boolean;  // 削除ボタンを表示するかどうか
  deleting?: boolean;   // 削除処理中かどうか（ボタンを無効化する）
  onDelete?: (videoId: string) => void; // 削除ボタンを押したときの処理
}

// -------------------------------------------------------
// 【課題 1】再生時間を「分:秒」の形式に変換する関数
//   引数: seconds（秒数、数値）
//   戻り値: "1:05" のような文字列。無効な値なら null を返す
// -------------------------------------------------------
function formatDuration(seconds: number | null | undefined): string | null {
  if (seconds == null || !Number.isFinite(seconds) || seconds < 0) return null;
  const m = /* ★ 何分かを計算しよう (Math.floor を使う) */ 0;
  const s = /* ★ 何秒かを計算しよう (Math.floor を使う) */ 0;
  // ★ `${m}:${s}` の形式で返そう。秒は必ず2桁にすること (padStart)
  return "";
}

// -------------------------------------------------------
// 【課題 2】視聴回数を読みやすい形式に変換する関数
//   10000 以上 → 「〇.〇万」
//   1000  以上 → 「〇.〇千」
//   それ以外   → そのまま数字を文字列で返す
// -------------------------------------------------------
function formatViewCount(count: number): string {
  if (count >= 10000) return /* ★ 「〇.〇万」の形式で返そう */ "";
  if (count >= 1000)  return /* ★ 「〇.〇千」の形式で返そう */ "";
  return `${count}`;
}

// -------------------------------------------------------
// 【課題 3】投稿日時を「〇分前」などの形式に変換する関数
//   現在時刻と投稿日時の差を計算して、適切な文字列を返す
// -------------------------------------------------------
function formatTimeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 60) return `${minutes}分前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return /* ★ 「〇時間前」の形式で返そう */ "";
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}日前`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}ヶ月前`;
  return `${Math.floor(months / 12)}年前`;
}

export default function VideoCard({
  video,
  canDelete = false,
  deleting = false,
  onDelete,
}: VideoCardProps) {
  const navigate = useNavigate();
  const [detectedDuration, setDetectedDuration] = useState<number | null>(null);
  const displayDuration = video.duration_seconds ?? detectedDuration;
  const durationLabel = formatDuration(displayDuration);

  return (
    // ★ id 属性を追加しよう: `video-card-${video.id}` という形式
    <div className="video-card fade-in" onClick={() => navigate(`/watch/${video.id}`)} id={`video-card-${video.id}`}>
      <div className="video-card__thumbnail">
        <VideoPreview
          videoId={video.id}
          onDurationLoaded={(seconds) =>
            setDetectedDuration((prev) => prev ?? seconds)
          }
        />

        {/* オーバーレイ: ユーザー名 + 再生時間 + アクション */}
        <div className="video-card__overlay">
          {/* ★ チャンネルページへ移動するリンク部分 */}
          <div className="video-card__overlay-top" onClick={(e) => { e.stopPropagation(); navigate(`/channel/${video.uploader.id}`); }}>
            <ChannelAvatar displayName={video.uploader.display_name} size={32} />
            {/* ★ 投稿者名を表示しよう (video.uploader.display_name) */}
            <span className="video-card__overlay-username">{video.uploader.display_name}</span>
          </div>
          <div className="video-card__overlay-bottom">
            {/* ★ durationLabel がある場合だけ再生時間を表示しよう */}
            {durationLabel && (
              <span className="video-card__duration">{durationLabel}</span>
            )}
          </div>
        </div>

        {/* 右サイドアクションボタン */}
        <div className="video-card__actions">
          {/* 削除ボタン（canDelete が true のときだけ表示）*/}
          {canDelete && onDelete && (
            <button
              className="video-card__action-btn video-card__action-btn--danger"
              onClick={(e) => {
                e.stopPropagation();
                void onDelete(video.id);
              }}
              disabled={deleting}
              title="削除"
            >
              🗑
            </button>
          )}
          {/* ★ いいねボタン: クリックしても動画ページに移動しないよう e.stopPropagation() を呼ぼう */}
          <button className="video-card__action-btn" onClick={(e) => e.stopPropagation()} title="いいね">
            ♡
          </button>
        </div>
      </div>

      {/* 動画タイトルと視聴情報 */}
      <div className="video-card__info">
        {/* ★ 動画タイトルを表示しよう (video.title) */}
        <h3 className="video-card__title">{video.title}</h3>
        <div className="video-card__meta">
          {/* ★ 視聴回数を formatViewCount で整形して表示しよう */}
          <span>{formatViewCount(video.view_count)} 回視聴</span>
          <span className="video-card__meta-dot" />
          {/* ★ 投稿日時を formatTimeAgo で整形して表示しよう */}
          <span>{formatTimeAgo(video.created_at)}</span>
        </div>
      </div>
    </div>
  );
}

export { formatDuration, formatViewCount, formatTimeAgo };

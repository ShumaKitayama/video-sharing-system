/* ===================================================
 * components/student/VideoGrid.tsx
 * [学生編集可] 動画カードのグリッドレイアウト
 * =================================================== */

// Video 型（動画データの型定義）と VideoCard コンポーネントをインポート
import type { Video } from "../../types";
import VideoCard from "./VideoCard";

// このコンポーネントが受け取る props（引数）の型定義
interface VideoGridProps {
  videos: Video[];   // 表示する動画の配列
  loading?: boolean; // 読み込み中かどうか（? = 省略可能）
  canDelete?: boolean; // 各カードに削除ボタンを表示するかどうか
  deleting?: boolean;  // 削除処理中かどうか（ボタンを無効化する）
  onDelete?: (videoId: string) => void; // 削除ボタンを押したときの処理
}

// -------------------------------------------------------
// SkeletonCard: データ読み込み中に表示するダミーカード
//   実際のデータの代わりにグレーのブロックを表示する。
//   CSS の skeleton クラスがキラキラアニメーションを担当。
// -------------------------------------------------------
function SkeletonCard() {
  return (
    <div className="video-card">
      {/* サムネイル部分のスケルトン */}
      <div className="skeleton skeleton--thumb" />
      <div className="video-card__info">
        {/* タイトル行のスケルトン */}
        <div className="skeleton skeleton--title" />
        {/* 視聴回数・日時行のスケルトン */}
        <div className="skeleton skeleton--text" />
      </div>
    </div>
  );
}

// -------------------------------------------------------
// VideoGrid: メインコンポーネント
//   loading / 空 / 通常 の3パターンで表示を切り替える
// -------------------------------------------------------
export default function VideoGrid({
  videos,
  loading,
  canDelete,
  deleting,
  onDelete,
}: VideoGridProps) {

  // ① 読み込み中：スケルトンカードを 8 枚並べる
  if (loading) {
    return (
      // video-grid クラス = CSS で 2 カラムのグリッドを定義済み
      <div className="video-grid">
        {/* Array.from({ length: 8 }) で長さ 8 の配列を作り、map で繰り返す */}
        {Array.from({ length: 8 }).map((_, i) => (
          // key は React がリストを効率よく更新するために必要な一意の値
          <SkeletonCard key={i} />
        ))}
      </div>
    );
  }

  // ② 動画が 0 件：空状態のメッセージを表示
  if (videos.length === 0) {
    return (
      // style 属性に直接オブジェクトを書くことでインラインスタイルを指定できる
      <div style={{ textAlign: "center", padding: "48px 0", color: "var(--text-secondary)" }}>
        <div style={{ fontSize: 48, marginBottom: 16 }}>📭</div>
        <p>動画が見つかりませんでした</p>
      </div>
    );
  }

  // ③ 通常表示：動画の配列を VideoCard として並べる
  return (
    <div className="video-grid">
      {/* videos 配列の各要素 (video) に対して VideoCard を1枚ずつ生成する */}
      {videos.map((video) => (
        // key には一意の ID を使う（index より安定しているため）
        <VideoCard
          key={video.id}
          video={video}
          canDelete={canDelete}
          deleting={deleting}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
}

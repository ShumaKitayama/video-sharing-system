/* ===================================================
 * components/student/CommentItem.tsx
 * [学生編集可] 個別コメント表示
 * コメント1件分のアバター・名前・日時・本文を表示する。
 * =================================================== */

// Comment 型（コメントデータの型定義）をインポート
import type { Comment } from "../../types";
// アバターアイコンコンポーネント
import ChannelAvatar from "./ChannelAvatar";
// VideoCard.tsx で定義した「〇分前」変換関数を再利用
import { formatTimeAgo } from "./VideoCard";

// このコンポーネントが受け取る props の型定義
interface CommentItemProps {
  comment: Comment; // 表示するコメントデータ1件
}

// -------------------------------------------------------
// CommentItem: コメント1件分の表示
//   [アバター] [投稿者名] [投稿日時]
//              [コメント本文]
//   という構造で表示する。
// -------------------------------------------------------
export default function CommentItem({ comment }: CommentItemProps) {
  return (
    // fade-in クラスで表示時にふわっとフェードインする
    <div className="comment fade-in">

      {/* 投稿者のアバターアイコン（サイズ 40px） */}
      <ChannelAvatar displayName={comment.author.display_name} size={40} />

      {/* コメント本体エリア */}
      <div className="comment__body">

        {/* ヘッダー行：投稿者名と投稿日時 */}
        <div className="comment__header">
          <span className="comment__author">{comment.author.display_name}</span>
          {/* formatTimeAgo で「3分前」「2日前」などに変換 */}
          <span className="comment__date">{formatTimeAgo(comment.created_at)}</span>
        </div>

        {/* コメント本文 */}
        <p className="comment__text">{comment.body}</p>

      </div>
    </div>
  );
}

/* ===================================================
 * components/student/CommentList.tsx
 * [学生編集可] コメント一覧表示
 * =================================================== */

import type { Comment } from "../../types";
import CommentItem from "./CommentItem";

interface CommentListProps {
  comments: Comment[];
  loading?: boolean;
}

export default function CommentList({ comments, loading }: CommentListProps) {
  return (
    <div className="comments">
      <div className="comments__header">
        <span className="comments__count">{comments.length} 件のコメント</span>
        <button className="comments__sort">🔽 並べ替え</button>
      </div>

      <div className="comment-input">
        <div className="comment-input__avatar">👤</div>
        <input
          className="comment-input__field"
          type="text"
          placeholder="コメントを追加..."
          id="comment-input"
        />
      </div>

      {loading ? (
        <div style={{ color: "var(--text-secondary)", padding: "16px 0" }}>読み込み中...</div>
      ) : (
        comments.map((comment) => (
          <CommentItem key={comment.id} comment={comment} />
        ))
      )}
    </div>
  );
}

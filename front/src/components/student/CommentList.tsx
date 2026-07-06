/* ===================================================
 * components/student/CommentList.tsx
 * [学生編集可] コメント一覧表示
 * コメントの件数・並び替えボタン・入力欄・一覧を表示する。
 * =================================================== */

// Comment 型（コメントデータの型定義）をインポート
import type { Comment } from "../../types";
// 1件分のコメントを表示する子コンポーネント
import CommentItem from "./CommentItem";

// このコンポーネントが受け取る props の型定義
interface CommentListProps {
  comments: Comment[]; // 表示するコメントの配列
  loading?: boolean;   // 読み込み中かどうか（? = 省略可能）
}

// -------------------------------------------------------
// CommentList: コメントセクション全体
//   ヘッダー → 入力欄 → コメント一覧 の順に表示する。
// -------------------------------------------------------
export default function CommentList({ comments, loading }: CommentListProps) {
  return (
    <div className="comments">

      {/* ===== ヘッダー：件数と並び替えボタン ===== */}
      <div className="comments__header">
        {/* comments.length で配列の要素数（コメント件数）を取得 */}
        <span className="comments__count">{comments.length} 件のコメント</span>
        <button className="comments__sort">🔽 並べ替え</button>
      </div>

      {/* ===== コメント入力欄 ===== */}
      {/* 現在は見た目だけで、送信機能は未実装 */}
      <div className="comment-input">
        <div className="comment-input__avatar">👤</div>
        <input
          className="comment-input__field"
          type="text"
          placeholder="コメントを追加..."
          id="comment-input" /* ← id を付けると JavaScript から要素を取得しやすい */
        />
      </div>

      {/* ===== コメント一覧 =====
           loading が true なら「読み込み中...」を表示、
           false なら comments 配列を map で展開して表示する。
           これを「条件（三項）演算子」と言う: 条件 ? 真の値 : 偽の値 */}
      {loading ? (
        <div style={{ color: "var(--text-secondary)", padding: "16px 0" }}>読み込み中...</div>
      ) : (
        comments.map((comment) => (
          // key には一意の ID を渡す（React がリストを効率よく管理するため）
          <CommentItem key={comment.id} comment={comment} />
        ))
      )}

    </div>
  );
}

/* ===================================================
 * components/student/CommentItem.tsx
 * [学生編集可] 個別コメント表示
 * =================================================== */

import type { Comment } from "../../types";
import ChannelAvatar from "./ChannelAvatar";
import { formatTimeAgo } from "./VideoCard";

interface CommentItemProps {
  comment: Comment;
}

export default function CommentItem({ comment }: CommentItemProps) {
  return (
    <div className="comment fade-in">
      <ChannelAvatar displayName={comment.author.display_name} size={40} />
      <div className="comment__body">
        <div className="comment__header">
          <span className="comment__author">{comment.author.display_name}</span>
          <span className="comment__date">{formatTimeAgo(comment.created_at)}</span>
        </div>
        <p className="comment__text">{comment.body}</p>
      </div>
    </div>
  );
}

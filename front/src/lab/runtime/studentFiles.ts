/* ===================================================
 * lab/runtime/studentFiles.ts
 * [保護] 生徒が編集できるファイルの一覧と「元のソースコード」
 *
 * `?raw` を付けて import すると、ファイルの中身が
 * ただの文字列として手に入る。Lab ではこれを
 * 「編集前のコード」として使う。
 * =================================================== */

import videoCardSource from "../../components/student/VideoCard.tsx?raw";
import videoGridSource from "../../components/student/VideoGrid.tsx?raw";
import commentItemSource from "../../components/student/CommentItem.tsx?raw";
import commentListSource from "../../components/student/CommentList.tsx?raw";
import channelAvatarSource from "../../components/student/ChannelAvatar.tsx?raw";
import indexCssSource from "../../styles/index.css?raw";

/** 部品（TSX）ファイルの名前。import "./VideoCard" の "VideoCard" 部分と同じ */
export const STUDENT_MODULE_NAMES = [
  "VideoCard",
  "VideoGrid",
  "CommentItem",
  "CommentList",
  "ChannelAvatar",
] as const;

export type StudentModuleName = (typeof STUDENT_MODULE_NAMES)[number];

/** 生徒が編集できるファイル（部品 + CSS） */
export const STUDENT_FILE_NAMES = [
  "VideoCard.tsx",
  "VideoGrid.tsx",
  "CommentItem.tsx",
  "CommentList.tsx",
  "ChannelAvatar.tsx",
  "index.css",
] as const;

export type StudentFileName = (typeof STUDENT_FILE_NAMES)[number];

/** ファイル名 → ソースコード の対応表 */
export type StudentSources = Record<StudentFileName, string>;

/** 編集前（リポジトリにある本物）のソースコード */
export const builtinSources: StudentSources = {
  "VideoCard.tsx": videoCardSource,
  "VideoGrid.tsx": videoGridSource,
  "CommentItem.tsx": commentItemSource,
  "CommentList.tsx": commentListSource,
  "ChannelAvatar.tsx": channelAvatarSource,
  "index.css": indexCssSource,
};

export function isStudentFileName(name: string): name is StudentFileName {
  return (STUDENT_FILE_NAMES as readonly string[]).includes(name);
}

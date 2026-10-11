/* ===================================================
 * components/internal/StudentSlots.tsx
 * [保護] 生徒の部品を「差し替え表」から取り出して表示する入口
 *
 * ページ（pages/）は components/student/ を直接 import せず、
 * ここを通して使う。こうすると Lab で生徒が書いた部品に
 * 入れ替えられるし、生徒の部品が壊れても本物で表示し直せる。
 * （表示用の関数 formatViewCount などは studentFormatters.ts）
 * =================================================== */

import { Component, type ComponentProps, type ComponentType, type ReactNode } from "react";
import type BuiltinVideoGrid from "../student/VideoGrid";
import type BuiltinVideoCard from "../student/VideoCard";
import type BuiltinCommentItem from "../student/CommentItem";
import type BuiltinCommentList from "../student/CommentList";
import type BuiltinChannelAvatar from "../student/ChannelAvatar";
import {
  getBuiltinModule,
  getStudentModule,
  useStudentRegistryVersion,
} from "../../lab/runtime/studentRegistry";
import type { StudentModuleName } from "../../lab/runtime/studentFiles";
import { reportStudentRuntimeError } from "../../lab/runtime/labMessages";

// -------------------------------------------------------
// エラーが起きたら本物の部品で表示し直す仕組み
// -------------------------------------------------------

interface StudentErrorBoundaryProps {
  file: string;
  fallback: ReactNode;
  children: ReactNode;
}

interface StudentErrorBoundaryState {
  hasError: boolean;
}

class StudentErrorBoundary extends Component<StudentErrorBoundaryProps, StudentErrorBoundaryState> {
  state: StudentErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(): StudentErrorBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(error: unknown) {
    console.error(`[StudentSlots] ${this.props.file} の表示でエラー`, error);
    reportStudentRuntimeError(this.props.file, error);
  }

  render() {
    return this.state.hasError ? this.props.fallback : this.props.children;
  }
}

/** 差し替え表から部品を取り出し、エラー対策をつけて表示する */
function StudentSlot<P extends object>({ name, props }: { name: StudentModuleName; props: P }) {
  // 部品が入れ替わると version が変わり、エラー状態もリセットされる
  const version = useStudentRegistryVersion();
  const Student = getStudentModule(name).default as ComponentType<P>;
  const Builtin = getBuiltinModule(name).default as ComponentType<P>;

  return (
    <StudentErrorBoundary key={version} file={`${name}.tsx`} fallback={<Builtin {...props} />}>
      <Student {...props} />
    </StudentErrorBoundary>
  );
}

// -------------------------------------------------------
// ページから使う部品（中身は差し替え表しだい）
// -------------------------------------------------------

export function VideoGrid(props: ComponentProps<typeof BuiltinVideoGrid>) {
  return <StudentSlot name="VideoGrid" props={props} />;
}

export function VideoCard(props: ComponentProps<typeof BuiltinVideoCard>) {
  return <StudentSlot name="VideoCard" props={props} />;
}

export function CommentItem(props: ComponentProps<typeof BuiltinCommentItem>) {
  return <StudentSlot name="CommentItem" props={props} />;
}

export function CommentList(props: ComponentProps<typeof BuiltinCommentList>) {
  return <StudentSlot name="CommentList" props={props} />;
}

export function ChannelAvatar(props: ComponentProps<typeof BuiltinChannelAvatar>) {
  return <StudentSlot name="ChannelAvatar" props={props} />;
}

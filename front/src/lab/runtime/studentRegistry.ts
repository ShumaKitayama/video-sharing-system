/* ===================================================
 * lab/runtime/studentRegistry.ts
 * [保護] 生徒の部品の「差し替え表」
 *
 * ページは生徒の部品を直接 import せず、この表から取り出す。
 * - ふだん     : 表には本物（リポジトリのファイル）が入っている
 * - Lab の中   : ブラウザで変換した「生徒が書いた版」に入れ替わる
 * =================================================== */

import { useSyncExternalStore } from "react";
import * as VideoCardModule from "../../components/student/VideoCard";
import * as VideoGridModule from "../../components/student/VideoGrid";
import * as CommentItemModule from "../../components/student/CommentItem";
import * as CommentListModule from "../../components/student/CommentList";
import * as ChannelAvatarModule from "../../components/student/ChannelAvatar";
import type { StudentModuleName } from "./studentFiles";

/** 1ファイル分の export（default や formatViewCount など） */
export type ModuleExports = Record<string, unknown>;

export type StudentModules = Record<StudentModuleName, ModuleExports>;

/** 本物の部品（壊れたときの避難先にもなる） */
export const builtinModules: StudentModules = {
  VideoCard: VideoCardModule,
  VideoGrid: VideoGridModule,
  CommentItem: CommentItemModule,
  CommentList: CommentListModule,
  ChannelAvatar: ChannelAvatarModule,
};

// いま使われている部品と、入れ替えた回数（画面の再描画に使う）
let currentModules: StudentModules = builtinModules;
let version = 0;
const listeners = new Set<() => void>();

export function getStudentModule(name: StudentModuleName): ModuleExports {
  return currentModules[name];
}

export function getBuiltinModule(name: StudentModuleName): ModuleExports {
  return builtinModules[name];
}

/** 部品をまとめて入れ替える。null を渡すと本物に戻る */
export function setStudentModules(modules: StudentModules | null): void {
  currentModules = modules ?? builtinModules;
  version += 1;
  listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getVersion(): number {
  return version;
}

/** 部品が入れ替わるたびに数字が増える。コンポーネントの再描画に使う */
export function useStudentRegistryVersion(): number {
  return useSyncExternalStore(subscribe, getVersion, getVersion);
}

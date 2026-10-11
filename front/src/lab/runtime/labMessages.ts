/* ===================================================
 * lab/runtime/labMessages.ts
 * [保護] Lab 画面（親）とプレビュー（iframe の中）のあいだの「手紙」
 *
 * 2つの画面は postMessage で手紙をやりとりする。
 * このファイルは軽いので、ふだんのサイトから import しても重くならない。
 * =================================================== */

import type { LabError } from "./compileStudent";

/** プレビュー用 iframe につける名前（ページを移動しても残る） */
export const PREVIEW_WINDOW_NAME = "lab-preview";

/** 無限ループなどで止まったことを見つけるための目印 */
export const APPLYING_FLAG_KEY = "lab:applying";

/** 親 → プレビュー */
export type ToPreviewMessage = {
  type: "lab:apply";
  /** ファイル名 → ソース。null なら全部を本物に戻す */
  sources: Record<string, string> | null;
};

/** プレビュー → 親 */
export type ToParentMessage =
  | { type: "lab:ready" }
  | { type: "lab:applied"; errors: LabError[] }
  | { type: "lab:runtime-error"; error: LabError };

/** いま動いているのが Lab のプレビューの中かどうか */
export function isLabPreview(): boolean {
  return window.name === PREVIEW_WINDOW_NAME && window.parent !== window;
}

export function postToParent(message: ToParentMessage): void {
  if (!isLabPreview()) return;
  window.parent.postMessage(message, window.location.origin);
}

/** 生徒の部品が表示中にエラーを出したことを親に知らせる */
export function reportStudentRuntimeError(file: string, err: unknown): void {
  const message = err instanceof Error ? err.message : String(err);
  postToParent({
    type: "lab:runtime-error",
    error: { file, message: `表示中にエラー: ${message}（元の部品で表示しています）` },
  });
}

/** sessionStorage は使えない環境もあるので、必ず try/catch で包む */
export function setApplyingFlag(on: boolean): void {
  try {
    if (on) sessionStorage.setItem(APPLYING_FLAG_KEY, "1");
    else sessionStorage.removeItem(APPLYING_FLAG_KEY);
  } catch {
    // 使えなくても Lab 自体は動く
  }
}

export function readApplyingFlag(): boolean {
  try {
    return sessionStorage.getItem(APPLYING_FLAG_KEY) === "1";
  } catch {
    return false;
  }
}

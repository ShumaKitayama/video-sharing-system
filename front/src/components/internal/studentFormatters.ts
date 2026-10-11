/* ===================================================
 * components/internal/studentFormatters.ts
 * [保護] 生徒の VideoCard.tsx にある「表示用の関数」の入口
 *
 * ページは formatViewCount などをここから使う。
 * Lab では生徒が書きかえた関数が使われ、
 * エラーが出たら本物の関数の結果を使う。
 * =================================================== */

import {
  formatDuration as builtinFormatDuration,
  formatTimeAgo as builtinFormatTimeAgo,
  formatViewCount as builtinFormatViewCount,
} from "../student/VideoCard";
import { getStudentModule } from "../../lab/runtime/studentRegistry";
import { reportStudentRuntimeError } from "../../lab/runtime/labMessages";

function callStudentFormatter<T>(
  exportName: string,
  builtin: (value: T) => string,
  value: T,
): string {
  const studentFn = getStudentModule("VideoCard")[exportName];
  if (typeof studentFn !== "function" || studentFn === builtin) return builtin(value);
  try {
    return String(studentFn(value));
  } catch (err) {
    reportStudentRuntimeError("VideoCard.tsx", err);
    return builtin(value);
  }
}

export function formatViewCount(count: number): string {
  return callStudentFormatter("formatViewCount", builtinFormatViewCount, count);
}

export function formatTimeAgo(dateStr: string): string {
  return callStudentFormatter("formatTimeAgo", builtinFormatTimeAgo, dateStr);
}

export function formatDuration(seconds: number | null | undefined): string | null {
  const studentFn = getStudentModule("VideoCard").formatDuration;
  if (typeof studentFn !== "function" || studentFn === builtinFormatDuration) {
    return builtinFormatDuration(seconds);
  }
  try {
    const result = studentFn(seconds);
    return result == null ? null : String(result);
  } catch (err) {
    reportStudentRuntimeError("VideoCard.tsx", err);
    return builtinFormatDuration(seconds);
  }
}

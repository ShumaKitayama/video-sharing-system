/* ===================================================
 * lab/runtime/compileStudent.ts
 * [保護] 生徒が書いた TSX を、ブラウザで動く JavaScript に変換する
 *
 * ふだんは Vite（パソコン上のツール）がやっている変換を、
 * Sucrase というライブラリを使ってブラウザの中で行う。
 * =================================================== */

import { transform } from "sucrase";

/** 生徒に見せるエラー */
export interface LabError {
  file: string;
  message: string;
  /** 何行目か（分かるときだけ） */
  line?: number;
}

export type CompileResult =
  | { ok: true; code: string }
  | { ok: false; error: LabError };

/**
 * 生徒の部品で使ってほしくない書き方。
 * 通信は hooks（useVideos など）にまかせるルールなので、ここで止める。
 */
const FORBIDDEN_PATTERNS: { pattern: RegExp; message: string }[] = [
  {
    pattern: /\bfetch\s*\(|\bXMLHttpRequest\b|\baxios\b/,
    message:
      "部品の中で通信（fetch など）は書けません。データは hooks（useVideos など）からもらいましょう。",
  },
  {
    pattern: /\beval\s*\(|\bnew\s+Function\b/,
    message: "eval や new Function は使えません。",
  },
];

/** "Unexpected token (12:5)" のような文字列から行番号を取り出す */
function findLineNumber(message: string): number | undefined {
  const match = message.match(/\((\d+):\d+\)/);
  return match ? Number(match[1]) : undefined;
}

export function compileStudentTsx(file: string, source: string): CompileResult {
  for (const rule of FORBIDDEN_PATTERNS) {
    if (rule.pattern.test(source)) {
      return { ok: false, error: { file, message: rule.message } };
    }
  }

  try {
    const result = transform(source, {
      transforms: ["typescript", "jsx", "imports"],
      jsxRuntime: "automatic",
      production: true,
      filePath: file,
    });
    return { ok: true, code: result.code };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return {
      ok: false,
      error: { file, message: `書き方にまちがいがあります: ${message}`, line: findLineNumber(message) },
    };
  }
}

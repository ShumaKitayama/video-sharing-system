/* ===================================================
 * lab/runtime/moduleLoader.ts
 * [保護] 変換した JavaScript を実行して、部品（export）を取り出す
 *
 * 生徒のコードの import は、下の「対応表」で決まったものだけ使える。
 *   import { useState } from "react"           → 本物の React
 *   import ChannelAvatar from "./ChannelAvatar" → 生徒が書いた ChannelAvatar
 *   import VideoPreview from "../internal/..."  → 本物の内部部品
 * =================================================== */

import * as React from "react";
import * as JsxRuntime from "react/jsx-runtime";
import { Link, NavLink, useLocation, useNavigate, useParams, useSearchParams } from "react-router-dom";
import VideoPreview from "../../components/internal/VideoPreview";
import { builtinModules, type ModuleExports, type StudentModules } from "./studentRegistry";
import { STUDENT_MODULE_NAMES, type StudentModuleName } from "./studentFiles";
import type { LabError } from "./compileStudent";

/** 外から借りるモジュール（__esModule は「ES モジュールです」という目印） */
const hostModules: Record<string, ModuleExports> = {
  react: { __esModule: true, default: React, ...React },
  "react/jsx-runtime": { __esModule: true, ...JsxRuntime },
  // ルーターは、生徒の部品で使いそうなものだけを貸す（全部だとサイトが重くなる）
  "react-router-dom": { __esModule: true, Link, NavLink, useLocation, useNavigate, useParams, useSearchParams },
  "../internal/VideoPreview": { __esModule: true, default: VideoPreview },
  // 型だけの import は変換で消えるが、念のため空っぽを返す
  "../../types": { __esModule: true },
};

function toStudentModuleName(specifier: string): StudentModuleName | null {
  const name = specifier.replace(/^\.\//, "").replace(/\.tsx?$/, "");
  return (STUDENT_MODULE_NAMES as readonly string[]).includes(name)
    ? (name as StudentModuleName)
    : null;
}

/**
 * 変換済みのコード一式を実行して、部品の表を作る。
 * - code が無いファイル（変換に失敗したもの）は本物を使う
 * - 実行中にエラーになったファイルも本物を使い、errors に記録する
 */
export function loadStudentModules(
  compiledCode: Partial<Record<StudentModuleName, string>>,
): { modules: StudentModules; errors: LabError[] } {
  const errors: LabError[] = [];
  const cache = new Map<StudentModuleName, ModuleExports>();

  function load(name: StudentModuleName): ModuleExports {
    const cached = cache.get(name);
    if (cached) return cached;

    const code = compiledCode[name];
    if (code === undefined) {
      cache.set(name, builtinModules[name]);
      return builtinModules[name];
    }

    // 先に空の箱を登録しておくと、おたがいを import していても止まらない
    const moduleObject = { exports: {} as ModuleExports };
    cache.set(name, moduleObject.exports);

    const require = (specifier: string): ModuleExports => {
      const studentName = toStudentModuleName(specifier);
      if (studentName) return load(studentName);
      const host = hostModules[specifier];
      if (host) return host;
      throw new Error(`"${specifier}" は import できません。`);
    };

    try {
      const run = new Function("require", "module", "exports", code);
      run(require, moduleObject, moduleObject.exports);
      cache.set(name, moduleObject.exports);
      return moduleObject.exports;
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      errors.push({ file: `${name}.tsx`, message: `実行中にエラー: ${message}` });
      cache.set(name, builtinModules[name]);
      return builtinModules[name];
    }
  }

  const modules = {} as StudentModules;
  for (const name of STUDENT_MODULE_NAMES) {
    modules[name] = load(name);
    if (typeof modules[name].default !== "function") {
      errors.push({
        file: `${name}.tsx`,
        message: "export default で部品（関数）を出してください。",
      });
      modules[name] = builtinModules[name];
    }
  }
  return { modules, errors };
}

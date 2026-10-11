/* ===================================================
 * lab/runtime/previewBridge.ts
 * [保護] プレビュー（iframe の中）で、親から届いたコードを動かす
 *
 * 1. 親から「このコードで表示して」という手紙が届く
 * 2. TSX を変換して部品を作る（compileStudent / moduleLoader）
 * 3. 差し替え表と CSS を入れ替える → 画面が描き直される
 * 4. 結果（エラーの有無）を親に返事する
 * =================================================== */

import { compileStudentTsx, type LabError } from "./compileStudent";
import { loadStudentModules } from "./moduleLoader";
import { setStudentModules } from "./studentRegistry";
import { setStudentCss } from "./studentCss";
import { STUDENT_MODULE_NAMES, type StudentModuleName } from "./studentFiles";
import {
  postToParent,
  setApplyingFlag,
  type ToPreviewMessage,
} from "./labMessages";

function applySources(sources: Record<string, string> | null): void {
  if (sources === null) {
    setStudentModules(null);
    setStudentCss(null);
    postToParent({ type: "lab:applied", errors: [] });
    return;
  }

  // 無限ループで止まったら、この目印が消えずに残る
  setApplyingFlag(true);

  const errors: LabError[] = [];
  const compiledCode: Partial<Record<StudentModuleName, string>> = {};

  for (const name of STUDENT_MODULE_NAMES) {
    const file = `${name}.tsx`;
    const source = sources[file];
    if (source === undefined) continue; // 本物を使う

    const result = compileStudentTsx(file, source);
    if (result.ok) compiledCode[name] = result.code;
    else errors.push(result.error);
  }

  const loaded = loadStudentModules(compiledCode);
  errors.push(...loaded.errors);

  setStudentModules(loaded.modules);
  setStudentCss(sources["index.css"] ?? null);

  // 描き直しが終わってから目印を消して、親に返事をする
  setTimeout(() => {
    setApplyingFlag(false);
    postToParent({ type: "lab:applied", errors });
  }, 0);
}

export function startPreviewBridge(): void {
  window.addEventListener("message", (event: MessageEvent<ToPreviewMessage>) => {
    // 自分と同じサイトの、親画面からの手紙だけを受け取る
    if (event.origin !== window.location.origin) return;
    if (event.source !== window.parent) return;
    if (event.data?.type !== "lab:apply") return;
    applySources(event.data.sources);
  });

  postToParent({ type: "lab:ready" });
}

/* ===================================================
 * main.tsx
 * アプリケーションのエントリーポイント
 * =================================================== */

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import StudentRoot from "./components/internal/StudentRoot";
import "./styles/index.css";
import { isLabPreview } from "./lab/runtime/labMessages";

// Lab のプレビュー（iframe の中）のときだけ、コードを受け取る仕組みを動かす
if (isLabPreview()) {
  void import("./lab/runtime/previewBridge").then((bridge) => bridge.startPreviewBridge());
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <StudentRoot />
  </StrictMode>
);

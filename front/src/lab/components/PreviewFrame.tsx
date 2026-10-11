/* ===================================================
 * lab/components/PreviewFrame.tsx
 * [保護] 本物のサイトを iframe で表示し、生徒のコードを送る
 * =================================================== */

import { useEffect, useEffectEvent, useRef } from "react";
import type { LabError } from "../runtime/compileStudent";
import { PREVIEW_WINDOW_NAME, type ToParentMessage, type ToPreviewMessage } from "../runtime/labMessages";
import type { StudentSources } from "../runtime/studentFiles";

interface PreviewFrameProps {
  /** 表示してほしいコード。null なら本物のまま */
  sources: StudentSources | null;
  /** コードが変わってから送るまでの待ち時間（打っている途中で何度も送らないため） */
  delayMs: number;
  /** この数字が変わると、待たずにすぐ送り直す（「▶ 実行」ボタン） */
  runToken: number;
  /** コードを送った直後に呼ばれる（前のエラーを消すため） */
  onSend: () => void;
  onApplied: (errors: LabError[]) => void;
  onRuntimeError: (error: LabError) => void;
}

export default function PreviewFrame({
  sources,
  delayMs,
  runToken,
  onSend,
  onApplied,
  onRuntimeError,
}: PreviewFrameProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const readyRef = useRef(false);
  const latestSourcesRef = useRef<StudentSources | null>(sources);
  const lastSentRef = useRef<string | null>(null);

  /** いまのコードを iframe に送る（同じ内容なら送らない） */
  const send = useEffectEvent((force: boolean) => {
    const target = iframeRef.current?.contentWindow;
    if (!target || !readyRef.current) return;

    const key = JSON.stringify(latestSourcesRef.current);
    if (!force && key === lastSentRef.current) return;
    lastSentRef.current = key;

    const message: ToPreviewMessage = { type: "lab:apply", sources: latestSourcesRef.current };
    target.postMessage(message, window.location.origin);
    onSend();
  });

  const handleMessage = useEffectEvent((event: MessageEvent<ToParentMessage>) => {
    if (event.origin !== window.location.origin) return;
    if (event.source !== iframeRef.current?.contentWindow) return;

    switch (event.data?.type) {
      case "lab:ready":
        // iframe が読み込み直されたら、最新のコードを送り直す
        readyRef.current = true;
        send(true);
        break;
      case "lab:applied":
        onApplied(event.data.errors);
        break;
      case "lab:runtime-error":
        onRuntimeError(event.data.error);
        break;
    }
  });

  useEffect(() => {
    const listener = (event: MessageEvent<ToParentMessage>) => handleMessage(event);
    window.addEventListener("message", listener);
    return () => window.removeEventListener("message", listener);
  }, []);

  // コードが変わったら、少し待ってから送る
  useEffect(() => {
    latestSourcesRef.current = sources;
    const timer = window.setTimeout(() => send(false), delayMs);
    return () => window.clearTimeout(timer);
  }, [sources, delayMs]);

  // 「▶ 実行」が押されたら、すぐ送る
  useEffect(() => {
    if (runToken > 0) send(true);
  }, [runToken]);

  return (
    <iframe
      ref={iframeRef}
      className="lab-preview__frame"
      name={PREVIEW_WINDOW_NAME}
      src="/"
      title="サイトのプレビュー"
    />
  );
}

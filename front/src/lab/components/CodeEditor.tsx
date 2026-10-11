/* ===================================================
 * lab/components/CodeEditor.tsx
 * [保護] 自由改造用のコードエディタ（CodeMirror 6）
 *
 * ファイルを切りかえるときは key を変えて作り直す前提。
 * =================================================== */

import { useEffect, useEffectEvent, useRef } from "react";
import { EditorView, basicSetup } from "codemirror";
import { javascript } from "@codemirror/lang-javascript";
import { css } from "@codemirror/lang-css";

interface CodeEditorProps {
  value: string;
  language: "tsx" | "css";
  onChange: (value: string) => void;
}

const editorTheme = EditorView.theme({
  "&": { height: "100%", fontSize: "14px", backgroundColor: "#ffffff" },
  ".cm-scroller": { fontFamily: "'Consolas', 'Menlo', monospace", lineHeight: "1.6" },
  ".cm-gutters": { backgroundColor: "#f7f4fd", borderRight: "1px solid #e5ddf5", color: "#a0a0b8" },
  ".cm-activeLine": { backgroundColor: "rgba(139, 92, 246, 0.06)" },
  ".cm-activeLineGutter": { backgroundColor: "rgba(139, 92, 246, 0.12)" },
});

export default function CodeEditor({ value, language, onChange }: CodeEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);

  const handleChange = useEffectEvent((text: string) => onChange(text));

  // 最初の1回だけエディタを作る
  useEffect(() => {
    if (!containerRef.current) return;
    const view = new EditorView({
      doc: value,
      parent: containerRef.current,
      extensions: [
        basicSetup,
        editorTheme,
        language === "css" ? css() : javascript({ jsx: true, typescript: true }),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) handleChange(update.state.doc.toString());
        }),
      ],
    });
    viewRef.current = view;
    return () => {
      view.destroy();
      viewRef.current = null;
    };
    // value は最初の中身としてだけ使う（変化は下の useEffect で反映する）
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [language]);

  // 外から中身が変わったとき（「元に戻す」など）だけ書きかえる
  useEffect(() => {
    const view = viewRef.current;
    if (!view) return;
    const current = view.state.doc.toString();
    if (current !== value) {
      view.dispatch({ changes: { from: 0, to: current.length, insert: value } });
    }
  }, [value]);

  return <div ref={containerRef} className="lab-editor" />;
}

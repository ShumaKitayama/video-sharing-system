/* ===================================================
 * lab/components/FreePanel.tsx
 * [保護] 自由改造の画面（ファイルのタブ + エディタ）
 * =================================================== */

import CodeEditor from "./CodeEditor";
import { STUDENT_FILE_NAMES, type StudentFileName } from "../runtime/studentFiles";

interface FreePanelProps {
  file: StudentFileName;
  source: string;
  editedFiles: StudentFileName[];
  onSelectFile: (file: StudentFileName) => void;
  onChangeSource: (source: string) => void;
  onRun: () => void;
  onResetFile: () => void;
}

export default function FreePanel({
  file,
  source,
  editedFiles,
  onSelectFile,
  onChangeSource,
  onRun,
  onResetFile,
}: FreePanelProps) {
  return (
    <div className="lab-free">
      <div className="lab-free__tabs">
        {STUDENT_FILE_NAMES.map((name) => (
          <button
            key={name}
            className={`lab-free__tab ${name === file ? "lab-free__tab--active" : ""}`}
            onClick={() => onSelectFile(name)}
          >
            {name}
            {editedFiles.includes(name) && <span className="lab-free__dot" title="書きかえ済み" />}
          </button>
        ))}
      </div>

      <div className="lab-free__editor">
        <CodeEditor
          key={file}
          value={source}
          language={file.endsWith(".css") ? "css" : "tsx"}
          onChange={onChangeSource}
        />
      </div>

      <div className="lab-free__actions">
        <button className="lab-btn lab-btn--ghost" onClick={onResetFile}>
          ↺ このファイルを元に戻す
        </button>
        <button className="lab-btn lab-btn--primary" onClick={onRun}>
          ▶ 実行
        </button>
      </div>
    </div>
  );
}

/* ===================================================
 * lab/components/ErrorList.tsx
 * [保護] 生徒のコードのエラーを、わかりやすく並べる
 * =================================================== */

import type { LabError } from "../runtime/compileStudent";

interface ErrorListProps {
  errors: LabError[];
}

export default function ErrorList({ errors }: ErrorListProps) {
  if (errors.length === 0) return null;

  return (
    <div className="lab-errors" role="alert">
      <div className="lab-errors__title">⚠️ コードにエラーがあります（そのファイルは元のコードで表示しています）</div>
      <ul>
        {errors.map((error, i) => (
          <li key={i} className="lab-errors__item">
            <span className="lab-errors__file">
              {error.file}
              {error.line !== undefined && ` の ${error.line} 行目`}
            </span>
            <span>{error.message}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

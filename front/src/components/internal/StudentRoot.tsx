/* ===================================================
 * components/internal/StudentRoot.tsx
 * [保護] 生徒の部品が入れ替わったら、アプリ全体を描き直す
 * （ふだんは入れ替わらないので、何もしないのと同じ）
 * =================================================== */

import App from "../../App";
import { useStudentRegistryVersion } from "../../lab/runtime/studentRegistry";

export default function StudentRoot() {
  // 入れ替わりの回数を読むだけで、変わったときに <App /> から描き直される
  useStudentRegistryVersion();
  return <App />;
}

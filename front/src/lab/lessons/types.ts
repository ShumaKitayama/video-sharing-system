/* ===================================================
 * lab/lessons/types.ts
 * レッスン（穴埋め問題）の形を決める型
 * =================================================== */

import type { StudentFileName } from "../runtime/studentFiles";

/**
 * 穴（ブランク）の種類
 * - choice : 選択肢から選ぶ（answer があれば正解判定する）
 * - text   : 自由に文字を入れる（空でなければOK）
 * - color  : カラーピッカーで色を選ぶ
 */
export type BlankKind = "choice" | "text" | "color";

/**
 * 入れた文字をコードに入れるときの「安全な変換」
 * - jsx-text : JSX の文章の中（{ } < > を全角にしてコードが壊れないようにする）
 * - string   : "..." の文字列の中（" と \ を逃がす）
 * - none     : そのまま（選択肢・色など、こちらで用意した値だけに使う）
 */
export type BlankEscape = "jsx-text" | "string" | "none";

export interface Blank {
  id: string;
  kind: BlankKind;
  /** 穴の上に出す短い説明（例: "くらべる記号"） */
  label: string;
  /** choice のときの選択肢 */
  options?: string[];
  /** choice の正解。書かなければどれを選んでも正解 */
  answer?: string;
  /** text / color の最初の値 */
  defaultValue?: string;
  escape?: BlankEscape;
  /** text の最大文字数（長すぎる文字で画面がこわれないように） */
  maxLength?: number;
}

/**
 * コードの書きかえ方。
 * find の文字列を探して、replace に置きかえる。
 * replace の中の {{id}} が穴になる。
 */
export interface CodeEdit {
  find: string;
  replace: string;
}

export interface LessonStep {
  id: string;
  title: string;
  /** やること（生徒に見せる説明） */
  description: string;
  /** 書きかえるファイル（自由改造ステップでは null） */
  file: StudentFileName | null;
  edits: CodeEdit[];
  blanks: Blank[];
  /** まちがえたときに出すヒント */
  hint?: string;
  /** 正解したあとに出す解説 */
  explanation?: string;
}

/* ===================================================
 * lab/labState.ts
 * [保護] Lab の進み具合の保存と、「いまのコード」の組み立て
 *
 * 考え方:
 *   本物のコード → 正解したステップの書きかえを順に当てる → (自由改造の書きかえ)
 * こうすると、何度やり直しても同じ結果になる。
 * =================================================== */

import type { Blank, LessonStep } from "./lessons/types";
import { builtinSources, isStudentFileName, type StudentFileName, type StudentSources } from "./runtime/studentFiles";

// -------------------------------------------------------
// 保存する情報
// -------------------------------------------------------

export type LabMode = "step" | "free";

export interface LabProgress {
  mode: LabMode;
  /** いま開いているステップの番号（0 から） */
  stepIndex: number;
  /** 正解したステップの id */
  completed: string[];
  /** ステップごとの穴の答え（stepId → blankId → 値） */
  answers: Record<string, Record<string, string>>;
  /** 自由改造で書きかえたファイル */
  freeEdits: Partial<StudentSources>;
}

const STORAGE_KEY = "lab:v1:progress";

export function createEmptyProgress(): LabProgress {
  return { mode: "step", stepIndex: 0, completed: [], answers: {}, freeEdits: {} };
}

/** localStorage は使えないこともあるので、必ず try/catch で包む */
export function loadProgress(): LabProgress {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return createEmptyProgress();
    const saved = JSON.parse(raw) as Partial<LabProgress>;
    const freeEdits: Partial<StudentSources> = {};
    for (const [file, source] of Object.entries(saved.freeEdits ?? {})) {
      if (isStudentFileName(file) && typeof source === "string") freeEdits[file] = source;
    }
    return {
      mode: saved.mode === "free" ? "free" : "step",
      stepIndex: typeof saved.stepIndex === "number" ? saved.stepIndex : 0,
      completed: Array.isArray(saved.completed) ? saved.completed : [],
      answers: saved.answers ?? {},
      freeEdits,
    };
  } catch {
    return createEmptyProgress();
  }
}

export function saveProgress(progress: LabProgress): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(progress));
  } catch {
    // 保存できなくても、その場では動く
  }
}

// -------------------------------------------------------
// 穴の値をコードに入れる
// -------------------------------------------------------

/** 入れた文字でコードが壊れないように変換する */
export function escapeBlankValue(blank: Blank, value: string): string {
  switch (blank.escape ?? "none") {
    case "jsx-text":
      return value
        .replace(/\{/g, "｛")
        .replace(/\}/g, "｝")
        .replace(/</g, "＜")
        .replace(/>/g, "＞")
        .replace(/\n/g, " ");
    case "string":
      return value.replace(/\\/g, "\\\\").replace(/"/g, '\\"').replace(/\n/g, " ");
    case "none":
      return value;
  }
}

/** 穴の最初の値（choice は未選択 = ""） */
export function defaultBlankValue(blank: Blank): string {
  return blank.kind === "choice" ? "" : (blank.defaultValue ?? "");
}

/** その穴に値が入っているか */
export function isBlankFilled(blank: Blank, value: string | undefined): boolean {
  if (value === undefined) return false;
  if (blank.kind === "choice") return blank.options?.includes(value) ?? false;
  if (blank.kind === "color") return /^#[0-9a-fA-F]{6}$/.test(value);
  return value.length <= (blank.maxLength ?? 40);
}

/** その穴の答えが正解か（answer が無ければ、入っていれば正解） */
export function isBlankCorrect(blank: Blank, value: string | undefined): boolean {
  if (!isBlankFilled(blank, value)) return false;
  return blank.answer === undefined || blank.answer === value;
}

function fillTemplate(template: string, blanks: Blank[], values: Record<string, string>): string {
  return template.replace(/\{\{(\w+)\}\}/g, (_, id: string) => {
    const blank = blanks.find((b) => b.id === id);
    if (!blank) return "";
    return escapeBlankValue(blank, values[id] ?? defaultBlankValue(blank));
  });
}

/** 1ステップ分の書きかえを当てる。find が見つからなければ null */
export function applyStepEdits(
  source: string,
  step: LessonStep,
  values: Record<string, string>,
): string | null {
  let result = source;
  for (const edit of step.edits) {
    const index = result.indexOf(edit.find);
    if (index === -1) return null;
    const replacement = fillTemplate(edit.replace, step.blanks, values);
    result = result.slice(0, index) + replacement + result.slice(index + edit.find.length);
  }
  return result;
}

// -------------------------------------------------------
// いまのコードを組み立てる
// -------------------------------------------------------

export interface BuildOptions {
  /** この番号より前のステップだけを使う（ステップ画面の「書きかえ前」用） */
  beforeStepIndex?: number;
  /** 正解前でも、このステップのこの答えをためしに当てる */
  draft?: { step: LessonStep; values: Record<string, string> };
  /** 自由改造の書きかえも入れるか */
  includeFreeEdits?: boolean;
}

export function buildSources(
  steps: LessonStep[],
  progress: LabProgress,
  options: BuildOptions = {},
): StudentSources {
  const sources: StudentSources = { ...builtinSources };
  const limit = options.beforeStepIndex ?? steps.length;

  steps.forEach((step, index) => {
    if (index >= limit || !step.file) return;
    const isDraft = options.draft?.step.id === step.id;
    if (!isDraft && !progress.completed.includes(step.id)) return;

    const values = isDraft ? options.draft!.values : (progress.answers[step.id] ?? {});
    const edited = applyStepEdits(sources[step.file], step, values);
    if (edited !== null) sources[step.file] = edited;
  });

  if (options.includeFreeEdits) {
    for (const [file, source] of Object.entries(progress.freeEdits)) {
      if (isStudentFileName(file) && typeof source === "string") sources[file] = source;
    }
  }
  return sources;
}

/** 自由改造で書きかえていないなら、ステップの結果のコードを返す */
export function getFreeEditSource(
  steps: LessonStep[],
  progress: LabProgress,
  file: StudentFileName,
): string {
  return progress.freeEdits[file] ?? buildSources(steps, progress)[file];
}

// -------------------------------------------------------
// ステップ画面に出すコード（穴の位置をわかるようにする）
// -------------------------------------------------------

export type CodeSegment = { type: "text"; text: string } | { type: "blank"; id: string };

export interface StepCodeView {
  /** 1行ずつ、文字と穴に分けたもの */
  lines: { number: number; segments: CodeSegment[] }[];
  /** find が見つからなかった（自由改造で消したなど） */
  missing: boolean;
}

const MARKER = "\u0000";
const CONTEXT_LINES = 3;

export function buildStepCodeView(baseSource: string, step: LessonStep): StepCodeView {
  let marked = baseSource;
  for (const edit of step.edits) {
    const index = marked.indexOf(edit.find);
    if (index === -1) return { lines: [], missing: true };
    const replacement = edit.replace.replace(/\{\{(\w+)\}\}/g, (_, id: string) => `${MARKER}${id}${MARKER}`);
    marked = marked.slice(0, index) + replacement + marked.slice(index + edit.find.length);
  }

  const allLines = marked.split("\n");
  const blankLineNumbers = allLines.flatMap((line, i) => (line.includes(MARKER) ? [i] : []));
  const first = Math.max(0, (blankLineNumbers[0] ?? 0) - CONTEXT_LINES);
  const last = Math.min(allLines.length - 1, (blankLineNumbers.at(-1) ?? 0) + CONTEXT_LINES);

  const lines = allLines.slice(first, last + 1).map((line, offset) => {
    // "abc\0id\0def" を split すると ["abc", "id", "def"] になり、奇数番目が穴
    const parts = line.split(MARKER);
    const segments: CodeSegment[] = parts.map((part, i) =>
      i % 2 === 1 ? { type: "blank", id: part } : { type: "text", text: part },
    );
    return { number: first + offset + 1, segments };
  });

  return { lines, missing: false };
}

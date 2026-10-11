/* ===================================================
 * lab/LabPage.tsx
 * [保護] 体験会用の Lab 画面（/lab）
 *
 * 左: ステップ（穴埋め）または 自由改造のエディタ
 * 右: 本物のサイトのプレビュー（生徒のコードで表示）
 * =================================================== */

import { useEffect, useMemo, useState } from "react";
import "./lab.css";
import { lessonSteps } from "./lessons/steps";
import {
  buildSources,
  createEmptyProgress,
  defaultBlankValue,
  getFreeEditSource,
  isBlankCorrect,
  isBlankFilled,
  loadProgress,
  saveProgress,
  type LabMode,
  type LabProgress,
} from "./labState";
import PreviewFrame from "./components/PreviewFrame";
import StepPanel, { type CheckStatus } from "./components/StepPanel";
import FreePanel from "./components/FreePanel";
import ErrorList from "./components/ErrorList";
import type { LabError } from "./runtime/compileStudent";
import { readApplyingFlag, setApplyingFlag } from "./runtime/labMessages";
import { STUDENT_FILE_NAMES, type StudentFileName } from "./runtime/studentFiles";

const STEP_PREVIEW_DELAY_MS = 250;
const FREE_PREVIEW_DELAY_MS = 800;

export default function LabPage() {
  const [progress, setProgress] = useState<LabProgress>(loadProgress);
  const [drafts, setDrafts] = useState<Record<string, Record<string, string>>>(() => progress.answers);
  const [status, setStatus] = useState<CheckStatus>("idle");
  const [freeFile, setFreeFile] = useState<StudentFileName>("VideoCard.tsx");
  const [runToken, setRunToken] = useState(0);
  const [errors, setErrors] = useState<LabError[]>([]);
  const [previewWide, setPreviewWide] = useState(false);
  // 前回プレビューが止まったまま閉じられたら、本物のサイトで始める
  const [safeMode, setSafeMode] = useState(readApplyingFlag);

  useEffect(() => {
    setApplyingFlag(false);
  }, []);

  useEffect(() => {
    saveProgress(progress);
  }, [progress]);

  const stepIndex = Math.min(Math.max(progress.stepIndex, 0), lessonSteps.length - 1);
  const step = lessonSteps[stepIndex];

  // いまのステップの穴の値（まだ触っていない穴は最初の値）
  const values = useMemo(() => {
    const result: Record<string, string> = {};
    for (const blank of step.blanks) {
      result[blank.id] = drafts[step.id]?.[blank.id] ?? defaultBlankValue(blank);
    }
    return result;
  }, [step, drafts]);

  // プレビューに送るコード
  const previewSources = useMemo(() => {
    if (safeMode) return null;
    if (progress.mode === "free") return buildSources(lessonSteps, progress, { includeFreeEdits: true });
    const allFilled = step.blanks.every((b) => isBlankFilled(b, values[b.id]));
    if (step.file && allFilled) return buildSources(lessonSteps, progress, { draft: { step, values } });
    return buildSources(lessonSteps, progress);
  }, [safeMode, progress, step, values]);

  // -------------------------------------------------------
  // ステップの操作
  // -------------------------------------------------------

  const goToStep = (index: number) => {
    setProgress((p) => ({ ...p, mode: "step", stepIndex: index }));
    setStatus("idle");
  };

  const handleChangeValue = (blankId: string, value: string) => {
    setDrafts((d) => ({ ...d, [step.id]: { ...values, ...d[step.id], [blankId]: value } }));
    setStatus("idle");
  };

  const handleCheck = () => {
    const correct = step.blanks.every((b) => isBlankCorrect(b, values[b.id]));
    if (!correct) {
      setStatus("wrong");
      return;
    }
    setProgress((p) => ({
      ...p,
      answers: { ...p.answers, [step.id]: values },
      completed: p.completed.includes(step.id) ? p.completed : [...p.completed, step.id],
    }));
    setStatus("correct");
  };

  const setMode = (mode: LabMode) => {
    setProgress((p) => ({ ...p, mode }));
    setStatus("idle");
  };

  const handleResetAll = () => {
    if (!window.confirm("ステップも自由改造も、全部最初にもどします。よいですか？")) return;
    setProgress(createEmptyProgress());
    setDrafts({});
    setStatus("idle");
    setErrors([]);
  };

  // -------------------------------------------------------
  // 自由改造の操作
  // -------------------------------------------------------

  const handleChangeFreeSource = (source: string) => {
    setProgress((p) => ({ ...p, freeEdits: { ...p.freeEdits, [freeFile]: source } }));
  };

  const handleResetFreeFile = () => {
    setProgress((p) => {
      const freeEdits = { ...p.freeEdits };
      delete freeEdits[freeFile];
      return { ...p, freeEdits };
    });
  };

  // 同じエラーが何度も届くことがあるので、重ならないように足していく
  const addErrors = (newErrors: LabError[]) => {
    setErrors((prev) => {
      const merged = [...prev];
      for (const error of newErrors) {
        if (!merged.some((e) => e.file === error.file && e.message === error.message)) merged.push(error);
      }
      return merged;
    });
  };

  const editedFiles = STUDENT_FILE_NAMES.filter((name) => progress.freeEdits[name] !== undefined);

  return (
    <div className="lab">
      <header className="lab-header">
        <div className="lab-header__logo">
          <span className="lab-header__logo-icon">🧪</span> VideoShare Lab
        </div>

        <div className="lab-header__modes">
          <button
            className={`lab-mode ${progress.mode === "step" ? "lab-mode--active" : ""}`}
            onClick={() => setMode("step")}
          >
            ステップ
          </button>
          <button
            className={`lab-mode ${progress.mode === "free" ? "lab-mode--active" : ""}`}
            onClick={() => setMode("free")}
          >
            自由改造
          </button>
        </div>

        <div className="lab-header__dots">
          {lessonSteps.map((s, i) => (
            <button
              key={s.id}
              className={[
                "lab-dot",
                progress.completed.includes(s.id) ? "lab-dot--done" : "",
                progress.mode === "step" && i === stepIndex ? "lab-dot--current" : "",
              ].join(" ")}
              title={`STEP ${i + 1}: ${s.title}`}
              onClick={() => goToStep(i)}
            >
              {i + 1}
            </button>
          ))}
        </div>

        <div className="lab-header__right">
          <button className="lab-btn lab-btn--ghost" onClick={() => setPreviewWide((w) => !w)}>
            {previewWide ? "◧ コードも見る" : "⛶ プレビューを大きく"}
          </button>
          <button className="lab-btn lab-btn--ghost" onClick={handleResetAll}>
            ↺ 最初から
          </button>
        </div>
      </header>

      {safeMode && (
        <div className="lab-banner">
          前回、コードが原因でプレビューが止まったため、元のサイトで表示しています。
          コードを直してから押してください →
          <button className="lab-btn lab-btn--primary" onClick={() => setSafeMode(false)}>
            もう一度ためす
          </button>
        </div>
      )}

      <div className={`lab-body ${previewWide ? "lab-body--wide" : ""}`}>
        <section className="lab-panel">
          {progress.mode === "step" ? (
            <StepPanel
              key={step.id}
              step={step}
              stepIndex={stepIndex}
              totalSteps={lessonSteps.length}
              baseSource={step.file ? buildSources(lessonSteps, progress, { beforeStepIndex: stepIndex })[step.file] : ""}
              values={values}
              status={status}
              completed={progress.completed.includes(step.id)}
              onChangeValue={handleChangeValue}
              onCheck={handleCheck}
              onPrev={() => goToStep(stepIndex - 1)}
              onNext={() => goToStep(stepIndex + 1)}
              onSkip={() => goToStep(Math.min(stepIndex + 1, lessonSteps.length - 1))}
              onStartFree={() => setMode("free")}
            />
          ) : (
            <FreePanel
              file={freeFile}
              source={getFreeEditSource(lessonSteps, progress, freeFile)}
              editedFiles={editedFiles}
              onSelectFile={setFreeFile}
              onChangeSource={handleChangeFreeSource}
              onRun={() => setRunToken((t) => t + 1)}
              onResetFile={handleResetFreeFile}
            />
          )}
          <ErrorList errors={errors} />
        </section>

        <section className="lab-preview">
          <div className="lab-preview__bar">
            <span className="lab-preview__dot" />
            <span className="lab-preview__dot" />
            <span className="lab-preview__dot" />
            <span className="lab-preview__label">プレビュー（あなたのコードで表示中）</span>
          </div>
          <PreviewFrame
            sources={previewSources}
            delayMs={progress.mode === "free" ? FREE_PREVIEW_DELAY_MS : STEP_PREVIEW_DELAY_MS}
            runToken={runToken}
            onSend={() => setErrors([])}
            onApplied={addErrors}
            onRuntimeError={(error) => addErrors([error])}
          />
        </section>
      </div>
    </div>
  );
}

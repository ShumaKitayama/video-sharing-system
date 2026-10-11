/* ===================================================
 * lab/components/StepPanel.tsx
 * [保護] 穴埋めステップの画面
 *
 * コードの穴（チップ）をクリック → 下で答えを選ぶ／入力する
 * → 「答え合わせ」で正解ならつぎのステップへ。
 * =================================================== */

import { useState } from "react";
import type { Blank, LessonStep } from "../lessons/types";
import { buildStepCodeView, isBlankCorrect, isBlankFilled } from "../labState";

export type CheckStatus = "idle" | "correct" | "wrong";

interface StepPanelProps {
  step: LessonStep;
  stepIndex: number;
  totalSteps: number;
  /** このステップの書きかえ前のコード */
  baseSource: string;
  values: Record<string, string>;
  status: CheckStatus;
  completed: boolean;
  onChangeValue: (blankId: string, value: string) => void;
  onCheck: () => void;
  onPrev: () => void;
  onNext: () => void;
  onSkip: () => void;
  onStartFree: () => void;
}

function chipText(blank: Blank, value: string | undefined): string {
  if (!value) return blank.kind === "choice" ? "？" : "（空っぽ）";
  return value;
}

export default function StepPanel({
  step,
  stepIndex,
  totalSteps,
  baseSource,
  values,
  status,
  completed,
  onChangeValue,
  onCheck,
  onPrev,
  onNext,
  onSkip,
  onStartFree,
}: StepPanelProps) {
  const [selectedId, setSelectedId] = useState<string | null>(step.blanks[0]?.id ?? null);
  const selected = step.blanks.find((b) => b.id === selectedId) ?? step.blanks[0];
  const codeView = buildStepCodeView(baseSource, step);
  const allFilled = step.blanks.every((b) => isBlankFilled(b, values[b.id]));
  const isLast = stepIndex === totalSteps - 1;

  return (
    <div className="lab-step">
      <div className="lab-step__head">
        <span className="lab-step__number">
          STEP {stepIndex + 1} / {totalSteps}
        </span>
        {completed && <span className="lab-step__done">✓ クリア</span>}
      </div>
      <h2 className="lab-step__title">{step.title}</h2>
      <p className="lab-step__desc">{step.description}</p>

      {step.file && (
        <>
          <div className="lab-code">
            <div className="lab-code__file">📄 {step.file}</div>
            {codeView.missing ? (
              <div className="lab-code__missing">
                自由改造でコードが変わっているため、このステップは表示できません。
                <button className="lab-btn lab-btn--ghost" onClick={onSkip}>
                  このステップを飛ばす
                </button>
              </div>
            ) : (
              <pre className="lab-code__body">
                {codeView.lines.map((line) => (
                  <div className="lab-code__line" key={line.number}>
                    <span className="lab-code__lineno">{line.number}</span>
                    <span className="lab-code__text">
                      {line.segments.map((segment, i) => {
                        if (segment.type === "text") return <span key={i}>{segment.text}</span>;
                        const blank = step.blanks.find((b) => b.id === segment.id);
                        if (!blank) return null;
                        const wrong = status === "wrong" && !isBlankCorrect(blank, values[blank.id]);
                        return (
                          <button
                            key={i}
                            className={[
                              "lab-chip",
                              selected?.id === blank.id ? "lab-chip--active" : "",
                              wrong ? "lab-chip--wrong" : "",
                              !values[blank.id] ? "lab-chip--empty" : "",
                            ].join(" ")}
                            onClick={() => setSelectedId(blank.id)}
                          >
                            {blank.kind === "color" && values[blank.id] && (
                              <span className="lab-chip__swatch" style={{ background: values[blank.id] }} />
                            )}
                            {chipText(blank, values[blank.id])}
                          </button>
                        );
                      })}
                    </span>
                  </div>
                ))}
              </pre>
            )}
          </div>

          {selected && !codeView.missing && (
            <BlankInput blank={selected} value={values[selected.id] ?? ""} onChange={onChangeValue} />
          )}
        </>
      )}

      {status === "wrong" && (
        <div className="lab-feedback lab-feedback--wrong">
          <strong>おしい！</strong> 赤い穴をもう一度見てみよう。
          {step.hint && <p>💡 {step.hint}</p>}
        </div>
      )}
      {status === "correct" && (
        <div className="lab-feedback lab-feedback--correct">
          <strong>🎉 正解！</strong> 右のプレビューを見てみよう。
          {step.explanation && <p>{step.explanation}</p>}
        </div>
      )}

      <div className="lab-step__actions">
        <button className="lab-btn lab-btn--ghost" onClick={onPrev} disabled={stepIndex === 0}>
          ◀ もどる
        </button>
        {step.file === null ? (
          <button className="lab-btn lab-btn--primary" onClick={onStartFree}>
            自由改造をはじめる ▶
          </button>
        ) : status === "correct" || (completed && status === "idle") ? (
          <button className="lab-btn lab-btn--primary" onClick={onNext} disabled={isLast}>
            つぎへ ▶
          </button>
        ) : (
          <button className="lab-btn lab-btn--primary" onClick={onCheck} disabled={!allFilled || codeView.missing}>
            答え合わせ
          </button>
        )}
      </div>
    </div>
  );
}

// -------------------------------------------------------
// 選んでいる穴に答えを入れる場所
// -------------------------------------------------------

interface BlankInputProps {
  blank: Blank;
  value: string;
  onChange: (blankId: string, value: string) => void;
}

function BlankInput({ blank, value, onChange }: BlankInputProps) {
  return (
    <div className="lab-answer">
      <div className="lab-answer__label">{blank.label}</div>

      {blank.kind === "choice" && (
        <div className="lab-answer__options">
          {blank.options?.map((option) => (
            <button
              key={option}
              className={`lab-option ${value === option ? "lab-option--active" : ""}`}
              onClick={() => onChange(blank.id, option)}
            >
              {option}
            </button>
          ))}
        </div>
      )}

      {blank.kind === "text" && (
        <input
          className="lab-answer__text"
          type="text"
          value={value}
          maxLength={blank.maxLength ?? 40}
          placeholder="ここに入力"
          onChange={(e) => onChange(blank.id, e.target.value)}
        />
      )}

      {blank.kind === "color" && (
        <div className="lab-answer__color">
          <input type="color" value={value || "#000000"} onChange={(e) => onChange(blank.id, e.target.value)} />
          <code>{value}</code>
        </div>
      )}
    </div>
  );
}

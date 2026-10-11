/* ===================================================
 * lab/runtime/studentCss.ts
 * [保護] Lab のプレビューで、生徒が書いた CSS に入れ替える
 *
 * ふだんは main.tsx が読み込む本物の index.css が効いている。
 * 生徒の CSS を使うときは、本物を「一時停止（disabled）」して
 * 代わりに <style id="student-css"> を入れる。
 * =================================================== */

const STYLE_ELEMENT_ID = "student-css";

/** 生徒の CSS 以外のスタイルシートを、止めたり動かしたりする */
function setOriginalStylesEnabled(enabled: boolean): void {
  for (const sheet of Array.from(document.styleSheets)) {
    const owner = sheet.ownerNode;
    if (owner instanceof HTMLElement && owner.id === STYLE_ELEMENT_ID) continue;
    sheet.disabled = !enabled;
  }
}

/** CSS の中身を入れ替える。null を渡すと本物の index.css に戻る */
export function setStudentCss(cssText: string | null): void {
  const existing = document.getElementById(STYLE_ELEMENT_ID);

  if (cssText === null) {
    existing?.remove();
    setOriginalStylesEnabled(true);
    return;
  }

  const style = existing ?? document.createElement("style");
  style.id = STYLE_ELEMENT_ID;
  style.textContent = cssText;
  if (!existing) document.head.appendChild(style);
  setOriginalStylesEnabled(false);
}

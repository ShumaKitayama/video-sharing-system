/* ===================================================
 * components/student/ChannelAvatar.tsx
 * [学生編集可] チャンネルアバターコンポーネント
 * ユーザー名の頭文字と、名前に基づいた固定色で
 * グラデーション丸アイコンを自動生成する。
 * =================================================== */

// このコンポーネントが受け取る props（引数）の型定義
interface ChannelAvatarProps {
  displayName: string; // 表示名（頭文字と色の決定に使う）
  size?: number;       // アイコンのサイズ（px）。省略すると 36px
  className?: string;  // 外から追加できる CSS クラス名
}

// -------------------------------------------------------
// アバターに使うグラデーションカラーのパレット
//   ユーザー名から自動で1色を選ぶので、同じ人は常に同じ色になる。
//   色を増やしたり変えたりしてみよう！
// -------------------------------------------------------
const avatarColors = [
  "linear-gradient(135deg, #667eea, #764ba2)", // 紫系
  "linear-gradient(135deg, #f093fb, #f5576c)", // ピンク系
  "linear-gradient(135deg, #4facfe, #00f2fe)", // 水色系
  "linear-gradient(135deg, #43e97b, #38f9d7)", // 緑系
  "linear-gradient(135deg, #fa709a, #fee140)", // オレンジピンク系
  "linear-gradient(135deg, #a18cd1, #fbc2eb)", // ラベンダー系
  "linear-gradient(135deg, #fccb90, #d57eeb)", // 黄紫系
  "linear-gradient(135deg, #89f7fe, #66a6ff)", // 青系
];

// -------------------------------------------------------
// getColorForName: 名前の文字コードからハッシュ値を計算し、
//   パレットの中から色を1つ選んで返す関数。
//   同じ名前なら必ず同じ色が返ってくる（決定論的）。
// -------------------------------------------------------
function getColorForName(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    // 文字コードとビットシフトでハッシュ値を計算
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  // 絶対値にして配列の長さで割った余り → 0〜7 のインデックス
  return avatarColors[Math.abs(hash) % avatarColors.length];
}

// -------------------------------------------------------
// ChannelAvatar: アバターアイコン本体
//   props から displayName・size・className を受け取る。
//   size のデフォルト値は 36、className のデフォルト値は ""。
// -------------------------------------------------------
export default function ChannelAvatar({ displayName, size = 36, className = "" }: ChannelAvatarProps) {
  // 頭文字（最初の1文字）を取り出す
  const initial = displayName.charAt(0);
  // 名前からグラデーションカラーを取得
  const bg = getColorForName(displayName);

  return (
    <div
      className={className}
      style={{
        width: size,           // props で受け取ったサイズ
        height: size,
        borderRadius: "50%",   // 正円にする
        background: bg,        // グラデーション背景
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontWeight: 600,
        fontSize: size * 0.4,  // サイズに比例したフォントサイズ（例: 36px → 14.4px）
        color: "#fff",         // 白文字
        flexShrink: 0,         // flex コンテナ内で縮まないように
      }}
    >
      {/* 頭文字を表示 */}
      {initial}
    </div>
  );
}

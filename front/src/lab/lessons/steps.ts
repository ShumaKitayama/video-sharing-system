/* ===================================================
 * lab/lessons/steps.ts
 * [先生が編集できる] 体験会のステップ（穴埋め問題）
 *
 * 書き方は lab/README.md を見てください。
 * find の文字列は components/student/ と styles/index.css の
 * 本物のコードに「そのまま」ある必要があります。
 * =================================================== */

import type { LessonStep } from "./types";

export const lessonSteps: LessonStep[] = [
  {
    id: "theme-color",
    title: "サイトの色を変えよう",
    description:
      "CSS の「変数」でサイト全体の色が決まっています。好きな色を選んで、右のプレビューの色が変わるのを見てみよう。",
    file: "index.css",
    edits: [
      { find: "--bg-primary: #f3eeff;", replace: "--bg-primary: {{bg}};" },
      { find: "--accent-purple: #8b5cf6;", replace: "--accent-purple: {{accent}};" },
    ],
    blanks: [
      { id: "bg", kind: "color", label: "背景の色", defaultValue: "#f3eeff", escape: "none" },
      { id: "accent", kind: "color", label: "アクセントの色", defaultValue: "#8b5cf6", escape: "none" },
    ],
    explanation:
      "--accent-purple のような「CSS 変数」は、いろいろな場所から var(--accent-purple) として使われています。1か所変えるだけで、サイト全体の色がそろって変わります。",
  },
  {
    id: "view-label",
    title: "「回視聴」の言葉を変えよう",
    description:
      "動画カードの下に出ている「〇〇 回視聴」の言葉は、VideoCard.tsx に書いてあります。好きな言葉に変えてみよう。",
    file: "VideoCard.tsx",
    edits: [
      {
        find: "<span>{formatViewCount(video.view_count)} 回視聴</span>",
        replace: "<span>{formatViewCount(video.view_count)} {{word}}</span>",
      },
    ],
    blanks: [
      { id: "word", kind: "text", label: "表示する言葉", defaultValue: "回視聴", escape: "jsx-text", maxLength: 20 },
    ],
    explanation:
      "{formatViewCount(video.view_count)} のように { } でかこまれた部分は「データ」、そのほかの部分はそのまま表示される「文字」です。",
  },
  {
    id: "new-badge",
    title: "if 文で「NEW」を出そう",
    description:
      "再生回数が 100 回より少ない動画は、数字のかわりに言葉を出すことにします。if 文の穴をうめよう。",
    file: "VideoCard.tsx",
    edits: [
      {
        find: "function formatViewCount(count: number): string {\n",
        replace:
          'function formatViewCount(count: number): string {\n  if (count {{op}} {{limit}}) return "{{label}}";\n',
      },
    ],
    blanks: [
      { id: "op", kind: "choice", label: "くらべる記号", options: [">=", "<", "==="], answer: "<", escape: "none" },
      { id: "limit", kind: "choice", label: "くらべる数", options: ["0", "100", "10000"], answer: "100", escape: "none" },
      { id: "label", kind: "text", label: "出す言葉", defaultValue: "NEW", escape: "string", maxLength: 10 },
    ],
    hint: "「100 回より少ない」は「count が 100 より小さい」ということ。小さいを表す記号はどれかな？",
    explanation:
      "if (条件) の条件が正しいときだけ return が動き、その下の行は実行されません。記号を >= にするとどうなるか、ためしてみるのもおすすめです。",
  },
  {
    id: "card-badge",
    title: "サムネイルにバッジを付けよう",
    description:
      "動画のサムネイルの下側に、小さなバッジを足します。バッジに表示する中身を選んでみよう。",
    file: "VideoCard.tsx",
    edits: [
      {
        find: '<div className="video-card__overlay-bottom">\n',
        replace:
          '<div className="video-card__overlay-bottom">\n            <span className="video-card__duration">{{content}}</span>\n',
      },
    ],
    blanks: [
      {
        id: "content",
        kind: "choice",
        label: "バッジの中身",
        options: ["⭐ おすすめ", "♡ {video.like_count}", "💬 {video.comment_count}"],
        escape: "none",
      },
    ],
    explanation:
      "「⭐ おすすめ」はどの動画でも同じ文字ですが、{video.like_count} を使うと動画ごとにちがう数（いいねの数）が出ます。データを表示するときは { } を使います。",
  },
  {
    id: "comment-name",
    title: "コメントの名前に「さん」を付けよう",
    description:
      "動画ページのコメント欄で、名前の前後に言葉を足してみよう（例: 後ろに「さん」）。",
    file: "CommentItem.tsx",
    edits: [
      {
        find: '<span className="comment__author">{comment.author.display_name}</span>',
        replace: '<span className="comment__author">{{before}}{comment.author.display_name}{{after}}</span>',
      },
    ],
    blanks: [
      { id: "before", kind: "text", label: "名前の前", defaultValue: "", escape: "jsx-text", maxLength: 10 },
      { id: "after", kind: "text", label: "名前の後ろ", defaultValue: "さん", escape: "jsx-text", maxLength: 10 },
    ],
    explanation:
      "CommentItem はコメント1つ分の部品です。1か所書きかえるだけで、全部のコメントの見た目が変わります。これが「部品（コンポーネント）」の便利なところです。",
  },
  {
    id: "free",
    title: "自由に改造しよう！",
    description:
      "ここからは「自由改造」モードで、ファイル全体を書きかえられます。色・文字・レイアウトを好きに変えて、自分だけのサイトにしよう。動画を投稿して友達と見せ合おう！",
    file: null,
    edits: [],
    blanks: [],
  },
];

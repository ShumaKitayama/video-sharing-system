/* ===================================================
 * mock/data.ts
 * バックエンド未実装時のデモ用サンプルデータ
 * =================================================== */

import type { Video, Comment, User } from "../types";

/** サンプルユーザー */
export const mockUsers: User[] = [
  { id: "u1", username: "yamada_taro", display_name: "山田 太郎", role: "student" },
  { id: "u2", username: "suzuki_hana", display_name: "鈴木 花子", role: "student" },
  { id: "u3", username: "tanaka_yuki", display_name: "田中 ゆき", role: "student" },
  { id: "u4", username: "sato_ken", display_name: "佐藤 健太", role: "student" },
  { id: "u5", username: "teacher", display_name: "先生", role: "teacher" },
  { id: "u6", username: "watanabe_ai", display_name: "渡辺 あい", role: "student" },
  { id: "u7", username: "takahashi_ryo", display_name: "高橋 涼", role: "student" },
  { id: "u8", username: "ito_miku", display_name: "伊藤 みく", role: "student" },
];

/** 動画タイトルをランダムに選ぶためのプール */
const videoEntries: Array<{
  title: string;
  description: string;
  duration: number;
  views: number;
  likes: number;
  comments: number;
  uploaderIdx: number;
}> = [
  {
    title: "水の状態変化の実験",
    description: "理科の授業で行った水の三態変化（固体・液体・気体）の実験を撮影しました。氷を加熱して蒸発するまでの温度変化を観察しています。",
    duration: 245,
    views: 1250,
    likes: 48,
    comments: 12,
    uploaderIdx: 0,
  },
  {
    title: "校庭で見つけた昆虫たち",
    description: "校庭の植え込みで見つけたいろいろな昆虫を紹介します！カマキリ、テントウムシ、チョウチョなどを間近で撮影しました。",
    duration: 382,
    views: 890,
    likes: 35,
    comments: 8,
    uploaderIdx: 1,
  },
  {
    title: "数学パズルに挑戦！",
    description: "数学の難問パズルを解いていく動画です。論理的な思考のプロセスを一緒に楽しみましょう。",
    duration: 605,
    views: 2340,
    likes: 102,
    comments: 23,
    uploaderIdx: 2,
  },
  {
    title: "美術の時間 - 水彩画テクニック",
    description: "水彩画で風景を描くテクニックを紹介します。グラデーションの作り方やにじみの使い方を解説。",
    duration: 480,
    views: 560,
    likes: 29,
    comments: 5,
    uploaderIdx: 3,
  },
  {
    title: "プログラミング入門 - HTMLで自己紹介ページ",
    description: "HTMLの基本タグを使って、自分だけの自己紹介Webページを作ってみよう！初心者向けに丁寧に解説しています。",
    duration: 720,
    views: 4500,
    likes: 210,
    comments: 45,
    uploaderIdx: 4,
  },
  {
    title: "給食のメニュー紹介",
    description: "今月の給食メニューの中から、特に美味しかったメニューをランキング形式で紹介します！",
    duration: 195,
    views: 3200,
    likes: 156,
    comments: 67,
    uploaderIdx: 5,
  },
  {
    title: "放課後の部活動 - バスケットボール",
    description: "放課後のバスケットボール部の練習風景です。シュート練習やミニゲームの様子を撮影しました。",
    duration: 540,
    views: 1800,
    likes: 89,
    comments: 15,
    uploaderIdx: 6,
  },
  {
    title: "英語の発音練習 - Rの発音",
    description: "英語のRとLの発音の違いを練習する動画です。ネイティブの発音と比較しながら練習してみましょう。",
    duration: 310,
    views: 980,
    likes: 42,
    comments: 9,
    uploaderIdx: 7,
  },
  {
    title: "理科実験 - 電池の仕組み",
    description: "レモンやジャガイモで電池を作る実験です。なぜ電気が発生するのかを解説しています。",
    duration: 425,
    views: 1540,
    likes: 68,
    comments: 18,
    uploaderIdx: 0,
  },
  {
    title: "音楽の授業 - リコーダー演奏",
    description: "リコーダーで「エーデルワイス」を演奏しました。運指のポイントも解説しています。",
    duration: 280,
    views: 720,
    likes: 31,
    comments: 7,
    uploaderIdx: 1,
  },
  {
    title: "社会科見学 - 市役所の仕事",
    description: "市役所を見学して、どんな仕事があるのかを取材しました。公務員の方にインタビューもしています。",
    duration: 660,
    views: 450,
    likes: 22,
    comments: 4,
    uploaderIdx: 2,
  },
  {
    title: "体育祭のダンス練習",
    description: "体育祭で踊るダンスの振り付けを練習している様子です。みんなで息を合わせて頑張っています！",
    duration: 350,
    views: 2100,
    likes: 95,
    comments: 28,
    uploaderIdx: 3,
  },
];

/** 色のプール（サムネイルの背景色として使用） */
const thumbnailColors = [
  "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
  "linear-gradient(135deg, #f093fb 0%, #f5576c 100%)",
  "linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)",
  "linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)",
  "linear-gradient(135deg, #fa709a 0%, #fee140 100%)",
  "linear-gradient(135deg, #a18cd1 0%, #fbc2eb 100%)",
  "linear-gradient(135deg, #fccb90 0%, #d57eeb 100%)",
  "linear-gradient(135deg, #e0c3fc 0%, #8ec5fc 100%)",
  "linear-gradient(135deg, #f5576c 0%, #ff6f91 100%)",
  "linear-gradient(135deg, #89f7fe 0%, #66a6ff 100%)",
  "linear-gradient(135deg, #fddb92 0%, #d1fdff 100%)",
  "linear-gradient(135deg, #96fbc4 0%, #f9f586 100%)",
];

/** 日付を過去のランダムな日にずらす */
function daysAgo(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString();
}

/** モック動画データを生成 */
export const mockVideos: Video[] = videoEntries.map((entry, idx) => ({
  id: `v${idx + 1}`,
  title: entry.title,
  description: entry.description,
  status: "published" as const,
  uploader: {
    id: mockUsers[entry.uploaderIdx].id,
    display_name: mockUsers[entry.uploaderIdx].display_name,
  },
  mime_type: "video/mp4",
  file_size_bytes: Math.floor(Math.random() * 100_000_000) + 5_000_000,
  duration_seconds: entry.duration,
  view_count: entry.views,
  like_count: entry.likes,
  comment_count: entry.comments,
  created_at: daysAgo(Math.floor(Math.random() * 60) + 1),
  updated_at: daysAgo(Math.floor(Math.random() * 30)),
  thumbnail_url: thumbnailColors[idx % thumbnailColors.length],
}));

/** モックコメントデータ */
export const mockComments: Record<string, Comment[]> = {
  v1: [
    { id: "c1", body: "すごく分かりやすかった！", author: { id: "u2", display_name: "鈴木 花子" }, created_at: daysAgo(5), updated_at: daysAgo(5) },
    { id: "c2", body: "水蒸気のところが面白い", author: { id: "u3", display_name: "田中 ゆき" }, created_at: daysAgo(4), updated_at: daysAgo(4) },
    { id: "c3", body: "もっと実験動画を見たい！", author: { id: "u4", display_name: "佐藤 健太" }, created_at: daysAgo(3), updated_at: daysAgo(3) },
  ],
  v3: [
    { id: "c4", body: "解き方が天才的！", author: { id: "u1", display_name: "山田 太郎" }, created_at: daysAgo(10), updated_at: daysAgo(10) },
    { id: "c5", body: "私も挑戦してみます", author: { id: "u5", display_name: "先生" }, created_at: daysAgo(9), updated_at: daysAgo(9) },
  ],
  v5: [
    { id: "c6", body: "HTMLが書けるようになった！", author: { id: "u6", display_name: "渡辺 あい" }, created_at: daysAgo(2), updated_at: daysAgo(2) },
    { id: "c7", body: "次はCSSもやってほしい", author: { id: "u7", display_name: "高橋 涼" }, created_at: daysAgo(1), updated_at: daysAgo(1) },
    { id: "c8", body: "初心者にもすごく分かりやすい", author: { id: "u8", display_name: "伊藤 みく" }, created_at: daysAgo(1), updated_at: daysAgo(1) },
    { id: "c9", body: "素晴らしい解説ですね", author: { id: "u2", display_name: "鈴木 花子" }, created_at: daysAgo(1), updated_at: daysAgo(1) },
  ],
  v6: [
    { id: "c10", body: "カレーが1位でしょ！", author: { id: "u1", display_name: "山田 太郎" }, created_at: daysAgo(7), updated_at: daysAgo(7) },
    { id: "c11", body: "揚げパン最高！", author: { id: "u3", display_name: "田中 ゆき" }, created_at: daysAgo(6), updated_at: daysAgo(6) },
  ],
};

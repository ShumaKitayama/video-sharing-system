# YouTube風UIフロントエンドの構築

`front/` ディレクトリに React + TypeScript + Vite で YouTube を模倣したプレミアムなダークテーマUIを構築する。

バックエンドAPIはまだ存在しないため、モックデータでデモ可能な状態にする。将来のAPI接続はサービス層の差し替えだけで完了する設計とする。

## Proposed Changes

### 1. プロジェクト初期化

#### [NEW] Vite + React + TypeScript プロジェクト

`front/` ディレクトリに `npx create-vite` でプロジェクトをスキャフォールディングする。

```bash
npx -y create-vite@latest ./ --template react-ts
npm install
npm install react-router-dom
```

---

### 2. ディレクトリ構成（AGENTS.md準拠）

```txt
front/
├── src/
│   ├── components/
│   │   ├── student/          ← [学生編集可] UIカスタマイズ用
│   │   │   ├── VideoCard.tsx        # サムネイルカード
│   │   │   ├── VideoGrid.tsx        # カードグリッドレイアウト
│   │   │   ├── CommentItem.tsx      # コメント表示
│   │   │   ├── CommentList.tsx      # コメント一覧
│   │   │   └── ChannelAvatar.tsx    # チャンネルアバター
│   │   └── internal/         ← [保護] インフラUI
│   │       ├── Header.tsx           # ヘッダーバー（検索・ナビ）
│   │       ├── Sidebar.tsx          # サイドバーナビゲーション
│   │       ├── VideoPlayer.tsx      # 動画プレーヤーラッパー
│   │       ├── UploadModal.tsx      # アップロードモーダル
│   │       ├── LoginModal.tsx       # ログインモーダル
│   │       └── Layout.tsx           # ページレイアウト骨格
│   ├── pages/
│   │   ├── HomePage.tsx             # トップページ（動画一覧）
│   │   ├── WatchPage.tsx            # 動画再生ページ
│   │   ├── SearchResultsPage.tsx    # 検索結果ページ
│   │   └── ChannelPage.tsx          # チャンネル（ユーザー）ページ
│   ├── services/              ← [保護] API通信の実装
│   │   └── api.ts                   # モック → 将来実APIに差し替え
│   ├── hooks/                 ← [保護] カスタムフック
│   │   ├── useVideos.ts             # 動画一覧取得
│   │   ├── useVideoDetail.ts        # 動画詳細取得
│   │   ├── useVideoUpload.ts        # 動画アップロード
│   │   ├── useComments.ts           # コメント取得・投稿
│   │   ├── useLike.ts               # いいね操作
│   │   └── useAuth.ts              # 認証状態管理
│   ├── api/                   ← [保護] エンドポイント定義
│   │   └── endpoints.ts
│   ├── types/                 ← 型定義
│   │   └── index.ts                 # Video, User, Comment等
│   ├── styles/                ← [学生編集可] スタイル
│   │   └── index.css                # グローバルCSS（YouTube風ダークテーマ）
│   ├── mock/                  ← モックデータ
│   │   └── data.ts
│   ├── App.tsx
│   └── main.tsx
├── index.html
├── package.json
├── tsconfig.json
└── vite.config.ts
```

---

### 3. デザインシステム（YouTube風ダークテーマ）

- **カラーパレット**: YouTubeダークモード風
  - 背景: `#0f0f0f` (メイン), `#1a1a1a` (カード), `#272727` (ホバー)
  - テキスト: `#f1f1f1` (メイン), `#aaaaaa` (セカンダリ)
  - アクセント: `#ff0000` (YouTubeレッド), `#065fd4` (リンク)
- **タイポグラフィ**: Google Fonts の `Roboto` (YouTube標準)
- **レスポンシブ**: モバイル〜デスクトップ対応グリッド
- **アニメーション**: ホバーエフェクト、フェードイン、スケルトンローディング

---

### 4. 主要ページの設計

#### HomePage（トップページ）
- ヘッダー: ロゴ、検索バー、ユーザーメニュー
- サイドバー: ホーム、急上昇、マイ動画、アップロード
- 動画グリッド: サムネイル + タイトル + チャンネル名 + 再生回数 + 投稿日時
- フィルターチップ: 「すべて」「最新」「人気」

#### WatchPage（再生ページ）
- 左: 動画プレーヤー + タイトル + アクションバー（いいね・共有）+ 説明 + コメント
- 右: 関連動画リスト（サイドバー）

#### SearchResultsPage（検索結果）
- 横長カード形式の検索結果一覧

#### UploadModal（アップロード）
- ドラッグ＆ドロップ対応
- プログレスバー
- タイトル・説明入力フォーム

---

### 5. モックデータ

バックエンド未実装のため、`src/mock/data.ts` にサンプル動画・ユーザー・コメントデータを用意する。サービス層が将来の実APIに切り替え可能な抽象化を維持する。

---

### 6. API抽象化レイヤー

```typescript
// services/api.ts - 現在はモック、将来は実APIに差し替え
export const videoService = {
  getVideos: async (params?) => { /* mock */ },
  getVideoById: async (id: string) => { /* mock */ },
  searchVideos: async (query: string) => { /* mock */ },
};
```

フック層がサービス層を呼び、学生コンポーネントがフックを使う3層構造で、学生はネットワーク処理に一切触れない。

---

## Verification Plan

### Automated Tests
```bash
cd front
npm run dev
```
ブラウザで `http://localhost:5173` にアクセスし、以下を確認:
- トップページに動画カードグリッドが表示される
- 動画カードクリックで再生ページに遷移する
- 検索バーが動作する
- サイドバーのナビゲーションが動作する
- アップロードモーダルが開く
- レスポンシブレイアウトが正しく動作する

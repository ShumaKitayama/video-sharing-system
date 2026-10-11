# front — VideoShare フロントエンド

React + TypeScript + Vite。Vercel に静的サイトとして配信し、`/api/*` は Vercel のバックエンドへ中継する（`vercel.json`）。

## 使い方は 2 通り

| 目的 | 方法 | 手順 |
| --- | --- | --- |
| **体験会（生徒）** | Chrome で `/lab` を開くだけ。インストール不要 | [`体験会手順.md`](./体験会手順.md) |
| 開発（先生・開発者） | PC でコードを編集して `npm run dev` / Docker | [`実行手順.md`](./実行手順.md), [`DOCKER.md`](./DOCKER.md) |

## フォルダ構成

| 場所 | 役割 | 生徒が編集 |
| --- | --- | --- |
| `src/components/student/` | 見た目の部品（VideoCard など） | ✅ |
| `src/styles/` | 色・CSS | ✅ |
| `src/components/internal/` | ヘッダー・プレイヤー・生徒部品の差し替え口（`StudentSlots.tsx`） | ❌ |
| `src/pages/` | ページの組み立て | ❌ |
| `src/hooks/`, `src/services/`, `src/api/` | 通信・状態管理 | ❌ |
| `src/lab/` | ブラウザ内 Lab（`/lab`）。ステップは `src/lab/lessons/steps.ts` | ❌（ステップは先生が編集可） |

Lab の仕組みとステップの書き方: [`src/lab/README.md`](./src/lab/README.md)

## コマンド

```bash
npm install
npm run dev      # http://localhost:5173 （/lab も同じサーバーで動く）
npm run build    # 型チェック + 本番ビルド
npm run lint
```

Vercel の API を使って開発するときは `VITE_REMOTE_BACKEND=true npm run dev`（詳しくは `vite.config.ts`）。

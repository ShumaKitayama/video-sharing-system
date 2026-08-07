# フロントエンドを Docker で起動する

> **まずは動かしたい人へ:** コマンドを上からコピーするだけの実行手順は [`実行手順.md`](./実行手順.md) にまとめてある。この文書は仕組みと詳細の説明である。

このページは、**フロントエンド（`front/`）だけを Docker で動かす**ための手順書である。

バックエンド（Go API）とデータベースは Vercel 上で動いているものを使うので、自分のパソコンで用意する必要はない。`docker compose` も使わない。

```txt
自分のパソコン                                  インターネット
┌────────────────────────┐
│ Docker コンテナ         │        ┌──────────────────────────────┐
│  Vite 開発サーバー       │ ─────▶ │ Vercel の API                 │
│  (localhost:5173)       │        │ video-sharing-api.vercel.app │
└────────────────────────┘        └──────────────────────────────┘
        ▲
        │ ブラウザで開く / src/ を編集すると即反映
```

---

## 1. 事前に必要なもの

- **Docker Desktop** がインストールされ、起動していること
  - 起動しているかの確認: ターミナルで `docker version` を実行してエラーが出なければ OK

Node.js のインストールは不要である（コンテナの中に入っている）。

---

## 2. 使うファイル（すでにこのリポジトリに入っている）

| ファイル | 役割 |
| --- | --- |
| `front/Dockerfile.dev` | コンテナの設計図。Node と依存パッケージを入れて Vite を起動する |
| `front/.dockerignore` | コンテナに持ち込まないファイルの一覧 |
| `front/vite.config.ts` | Vercel の API へ中継する dev proxy の設定 |

新しく作るファイルは無い。

---

## 3. 起動手順

### 手順 1: `front` フォルダに移動する

ターミナルを開き、このリポジトリの `front` フォルダへ移動する。

```bash
cd front
```

> パスがわからない場合は、`cd ` と入力したあとに Finder から `front` フォルダをターミナルへドラッグ&ドロップすると自動でパスが入る。

### 手順 2: イメージを作る（初回と、package.json を変えたときだけ）

```bash
npm run docker:build
```

初回は 1〜3 分ほどかかる。`naming to docker.io/library/video-front-dev` のような行が出れば成功である。

### 手順 3: 起動する

```bash
npm run docker:dev
```

次のような表示が出れば起動成功である。

```txt
  VITE v8.0.13  ready in 512 ms

  ➜  Local:   http://localhost:5173/
  ➜  Network: http://172.17.0.2:5173/
```

ブラウザで **http://localhost:5173** を開く。

> `Network:` に出る `172.x.x.x` は Docker 内部のアドレスなので使わない。必ず `localhost:5173` を開くこと。

### 手順 4: 止める

起動したターミナルで `Ctrl + C` を押す。コンテナは自動的に削除される（`--rm` を付けているため）。

---

## 4. コードを編集する

`front/src/` の中を編集して保存すると、**ブラウザが自動で更新される**（HMR）。コンテナを再起動する必要はない。

生徒が編集してよい場所:

- `front/src/components/student/` … 見た目のコンポーネント
- `front/src/styles/` … 色や CSS

> `front/vite.config.ts` や `front/package.json` を変更したときだけ、`Ctrl + C` で止めて手順 2 からやり直す。

---

## 5. Windows（PowerShell）で使う場合

`npm run docker:dev` は macOS / Linux 用の書き方なので、Windows では次のコマンドを直接実行する。

```powershell
docker build -f Dockerfile.dev -t video-front-dev .
docker run --rm -it -p 5173:5173 -v "${PWD}:/app" -v /app/node_modules video-front-dev
```

---

## 6. 接続先のバックエンドを変える

既定では `https://video-sharing-api.vercel.app` に接続する。別のサーバーを使いたい場合は `-e` で上書きする。

```bash
docker run --rm -it -p 5173:5173 \
  -e VITE_REMOTE_BACKEND_URL=https://別のサーバー.example.com \
  -v "$PWD":/app -v /app/node_modules video-front-dev
```

---

## 7. 仕組みの説明（先生・興味がある人向け）

### なぜ dev proxy を使うのか

ブラウザから Vercel の API を直接呼ぶと、**ログインが維持できない**。

API が発行するログイン Cookie は `SameSite=Lax` である。この設定の Cookie は「別サイトへのリクエスト」には送られない。`http://localhost:5173`（フロント）と `https://video-sharing-api.vercel.app`（API）は別サイトなので、ログインしても次のリクエストで未ログインに戻ってしまう。

そこで Vite 開発サーバー自身が `/api/*` を API へ中継する。ブラウザから見ると API はフロントと同じ `localhost:5173` になり、Cookie が正しく送られる。これは本番（Vercel）で `front/vercel.json` の rewrite がやっていることと同じである。

あわせて、中継の際に Cookie の `Secure` 属性を外している。`Secure` が付いた Cookie は HTTPS 用で、`http://localhost` では保存を拒否するブラウザがあるためである。

### なぜアップロード方式が切り替わるのか

Vercel の API には「リクエストの中身は約 4.5MB まで」という上限がある。動画をそのまま API に送ると大きすぎて弾かれる。

そのため Docker 起動時は、本番と同じ **Vercel Blob への直接アップロード方式** に切り替わる。動画本体は API を通らず、ブラウザから Vercel Blob へ直接送られる。

この切り替えは `VITE_REMOTE_BACKEND=true`（`Dockerfile.dev` で設定済み）が担っており、`front/src/api/endpoints.ts` と `front/src/hooks/useVideoUpload.ts` が参照している。

### 自分のパソコンで Go API も動かしたい場合

その場合は Docker を使わず、これまでどおりの手順で起動する。挙動は今までと変わっていない。

```bash
# ターミナル1
cd server && go run ./cmd/api

# ターミナル2
cd front && npm run dev
```

---

## 8. うまくいかないときは

| 症状 | 原因と対処 |
| --- | --- |
| `Cannot connect to the Docker daemon` | Docker Desktop が起動していない。アプリを起動してから再実行する |
| ビルドが `load metadata for docker.io/library/node:20-alpine` から進まない | Docker Desktop のログイン情報ヘルパー（`docker-credential-desktop`）が応答していない。Docker Desktop を再起動する。急ぐ場合は下の「応急処置」を使う |
| `port is already allocated` | 5173 番ポートを他のプロセスが使っている。すでに `npm run dev` が動いていないか確認して止める |
| ブラウザが真っ白 | `localhost:5173` ではなく `172.x.x.x:5173` を開いていないか確認する |
| 保存しても画面が変わらない | `front/src/` の外を編集していないか確認する。設定ファイルを変えた場合は再ビルドが必要 |
| ログインしてもすぐログアウトされる | `Ctrl + C` で止めてから `npm run docker:build` をやり直す（proxy 設定が古い可能性がある） |
| 動画のアップロードが失敗する | ログインしているか確認する。ファイルは MP4 / WebM、500MB 以下である必要がある |

### 応急処置: ビルドが固まって進まないとき

Docker Desktop を再起動しても直らない場合、ログイン情報を使わない一時設定でビルドできる。公開イメージの取得にログインは不要なため、これで問題なく動く。

```bash
mkdir -p /tmp/docker-nocreds && echo '{}' > /tmp/docker-nocreds/config.json
DOCKER_CONFIG=/tmp/docker-nocreds docker build -f Dockerfile.dev -t video-front-dev .
```

`npm run docker:dev` はそのまま使える（起動時はイメージの取得が不要なため）。

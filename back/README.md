# バックエンド設計ドキュメント

このディレクトリは、動画共有システムのバックエンド実装前に合意しておくべき設計をまとめる場所である。  
実装コードは `AGENTS.md` の方針どおり将来的に `server/` 配下へ置く想定とし、`back/` は設計資料専用の入口として使う。

---

## 1. 設計の目的

この設計は、次の条件を同時に満たすことを目的とする。

1. 中学生にも処理の流れを説明しやすい
2. 教室内LANで安定して動く
3. 動画アップロードと再生でメモリを使いすぎない
4. フロントエンドの学生編集領域と、内部処理を明確に分離する
5. 後からGo + Gin + PostgreSQLで素直に実装できる

---

## 2. 設計方針

### 2.1 教育用に単純な責務分離

バックエンドは次の4層を基本にする。

```txt
handler      HTTP入力とHTTP出力だけを扱う
service      業務ルールとトランザクション境界を扱う
repository   SQLだけを扱う
storage      ファイル保存と読み出しだけを扱う
```

これにより、たとえば動画アップロードは次のように説明できる。

```txt
1. handler が multipart/form-data を受け取る
2. service が入力値と権限を確認する
3. storage が動画をディスクへストリーム保存する
4. repository が動画メタデータをDBへ保存する
5. handler がJSONを返す
```

### 2.2 ローカルLAN向けの現実的な範囲

- 認証は複雑なJWTではなく、HTTP Only Cookieの単純なセッション方式にする
- 動画ファイルはローカルディスクに保存する
- 動画配信はMP4/WebMをそのままRange配信する
- クラウドストレージ、マイクロサービス、分散キャッシュは採用しない
- 先生権限と生徒権限だけを用意し、権限モデルを増やしすぎない

### 2.3 メモリ安全性

動画は大きいので、次の禁止事項を設計段階から守る。

- 動画全体を `io.ReadAll` で読む設計にしない
- `os.ReadFile` で動画を丸ごとメモリに載せない
- 動画サイズに比例した巨大バッファを確保しない

推奨する処理は次のとおりである。

- アップロードは固定長バッファを使った逐次コピー
- 再生はHTTP Range Request対応
- ファイルハンドルは必ず閉じる
- 失敗時は途中生成ファイルを掃除する

---

## 3. 用意する機能

### 3.1 必須機能

| 分類 | 機能 |
| --- | --- |
| 稼働監視 | ヘルスチェック、DB接続確認 |
| 認証 | ログイン、ログアウト、自分情報取得 |
| 利用者 | 生徒/先生ユーザーの作成、一覧、更新、削除 |
| 動画 | アップロード、一覧、詳細、編集、削除、公開制御 |
| 再生 | Range対応ストリーミング |
| コメント | 作成、一覧、削除 |
| 反応 | いいね追加、解除、自分の状態確認 |
| 集計 | 再生回数、いいね数、コメント数 |

### 3.2 あると便利な機能

| 分類 | 機能 |
| --- | --- |
| 検索 | タイトル、説明文、投稿者名での動画検索 |
| 絞り込み | 投稿者、自分の動画、公開状態、作成日時 |
| 管理 | 先生による動画非表示、ユーザー無効化 |
| 保守 | 期限切れセッション削除、孤立ファイル点検 |

### 3.3 今回は入れない機能

以下は教育目的に対して複雑さが先に増えるため、初期設計から除外する。

- JWTリフレッシュトークン
- OAuth / SNSログイン
- HLS / DASH / トランスコード基盤
- AWS S3など外部ストレージ
- 高度なレコメンド
- 通知基盤
- リアルタイムチャット
- 多段階のモデレーションワークフロー

---

## 4. ドキュメント一覧

| ファイル | 内容 |
| --- | --- |
| [`api-design.md`](./api-design.md) | REST API、認証、エラー、各エンドポイント仕様 |
| [`database-design.md`](./database-design.md) | テーブル、制約、ER関係、DDL、インデックス |
| [`query-catalog.md`](./query-catalog.md) | 実装で使うSQLクエリ一覧、トランザクション例 |

---

## 5. 想定するバックエンド構成

```txt
server/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── handler/
│   ├── service/
│   ├── repository/
│   ├── storage/
│   ├── streaming/
│   └── middleware/
├── migrations/
├── uploads/
│   └── videos/
└── go.mod
```

### 5.1 ディレクトリ責務

| ディレクトリ | 責務 |
| --- | --- |
| `handler/` | ルーティング、バインド、HTTPステータス |
| `service/` | 権限、入力検証、ユースケース |
| `repository/` | SQL実行、行の読み書き |
| `storage/` | 保存名生成、ファイル保存、削除 |
| `streaming/` | Range処理、Content-Type、部分配信 |
| `middleware/` | 認証、リクエストID、ログ |

---

## 6. データの大まかな流れ

### 6.1 動画アップロード

```txt
Browser
  ↓ multipart/form-data
handler.UploadVideo
  ↓
service.UploadVideo
  ├─ 入力検証
  ├─ 権限確認
  ├─ storage.SaveVideoStream
  └─ repository.CreateVideo
  ↓
JSONレスポンス
```

### 6.2 動画再生

```txt
Browser video element
  ↓ Range: bytes=...
handler.StreamVideo
  ↓
service.GetPlayableVideo
  ↓
repository.FindVideoStorageInfo
  ↓
streaming.ServeRange
  ↓
206 Partial Content / 200 OK
```

### 6.3 コメント投稿

```txt
Browser
  ↓ JSON
handler.CreateComment
  ↓
service.CreateComment
  ├─ 動画存在確認
  ├─ コメント本文検証
  └─ repository.CreateCommentWithCounter
  ↓
JSONレスポンス
```

---

## 7. 実装順序の推奨

1. `health` / `ready`
2. `users` と `sessions`
3. `videos` のメタデータCRUD
4. 動画ファイル保存
5. 動画Range配信
6. コメント
7. いいね
8. 検索と管理用フィルタ

この順なら、早い段階で「アップロードして再生する」最小体験まで到達できる。

---

## 8. 実装時の固定ルール

### 8.1 API

- ベースパスは `/api/v1`
- 公開IDはUUID
- 日時はRFC3339
- 成功レスポンスは `data`
- 一覧レスポンスは `meta`
- 失敗レスポンスは `error`

### 8.2 DB

- 内部主キーは `BIGSERIAL`
- 外部公開用IDは `UUID`
- 論理削除対象は `deleted_at`
- 参照整合性は外部キーで守る
- 集計カラムはトランザクション内で更新する

### 8.3 ファイル

- DBには絶対パスを持たせない
- DBには相対的な `storage_key` を保存する
- 元ファイル名は表示用として別カラムに持つ
- 保存名は衝突しないUUIDベースにする

---

## 9. 仕様の読み方

実装時は次の順で読むと迷いにくい。

1. `README.md` で全体像を見る
2. `api-design.md` で外から見える振る舞いを決める
3. `database-design.md` で保存構造を決める
4. `query-catalog.md` でrepository層に落とす

この順序なら、HTTP仕様とDB仕様がずれにくい。

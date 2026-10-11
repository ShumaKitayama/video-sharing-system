# src/lab — ブラウザ内 Lab（体験会モード）

`/lab` を開くと、左に穴埋めステップ／コードエディタ、右に本物のサイトのプレビューが出る。
生徒のコードはブラウザの中で変換され、`components/student/` の部品だけが差し替わる。

## フォルダ構成

| 場所 | 役割 | 編集 |
| --- | --- | --- |
| `lessons/steps.ts` | 体験会のステップ（穴埋め問題） | **先生が編集してよい** |
| `lessons/types.ts` | ステップの型 | 保護 |
| `LabPage.tsx`, `components/`, `lab.css` | Lab 画面 | 保護 |
| `labState.ts` | 進み具合の保存・コードの組み立て | 保護 |
| `runtime/` | 変換（Sucrase）・差し替え表・iframe との通信 | 保護 |

関係する保護ファイル:

- `components/internal/StudentSlots.tsx` … ページはここから生徒の部品を使う（エラー時は本物で表示）
- `components/internal/studentFormatters.ts` … `formatViewCount` などの入口
- `components/internal/StudentRoot.tsx` … 部品が差し替わったら全体を描き直す

## ステップの書き方

```ts
{
  id: "view-label",                 // ほかと重ならない名前（保存に使うので後から変えない）
  title: "「回視聴」の言葉を変えよう",
  description: "生徒に見せる説明",
  file: "VideoCard.tsx",            // 書きかえるファイル（自由改造ステップは null）
  edits: [
    {
      // 本物のコードに「そのまま」ある文字列（改行やスペースも一致させる）
      find: "<span>{formatViewCount(video.view_count)} 回視聴</span>",
      // 置きかえ後。{{id}} が穴になる
      replace: "<span>{formatViewCount(video.view_count)} {{word}}</span>",
    },
  ],
  blanks: [
    { id: "word", kind: "text", label: "表示する言葉", defaultValue: "回視聴", escape: "jsx-text", maxLength: 20 },
  ],
  hint: "まちがえたときのヒント（なくてもよい）",
  explanation: "正解したときの解説（なくてもよい）",
}
```

### 穴の種類（`kind`）

| kind | 使い方 | 正解判定 |
| --- | --- | --- |
| `choice` | `options` から選ぶ | `answer` があれば一致で正解。なければどれでも正解 |
| `text` | 自由入力 | 入っていれば正解（`maxLength` まで） |
| `color` | カラーピッカー | `#rrggbb` なら正解 |

### `escape`（入力でコードが壊れないようにする）

| escape | 使う場所 | 変換 |
| --- | --- | --- |
| `jsx-text` | JSX の文章の中 | `{ } < >` を全角にする |
| `string` | `"..."` の中 | `"` と `\` を逃がす |
| `none` | 選択肢・色など、こちらで決めた値 | そのまま |

## 注意

- `find` は**元のコードと 1 文字でもちがうと見つからない**。`components/student/` を書きかえたら、
  ステップの `find` も合わせて直すこと
- 2 つのステップが同じ場所を書きかえないようにする（前のステップの結果に対して `find` を探すため）
- `components/student/` から import できるのは `react`・`react-router-dom` の一部・生徒の部品同士・
  `../internal/VideoPreview` だけ。増やすときは `runtime/moduleLoader.ts` の `hostModules` に足す

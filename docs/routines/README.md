# Routines

Claude Code on the web の Routines（scheduled tasks）で動かす 4 本のプロンプト。
各ファイルの「プロンプト」節を [claude.ai/code/routines](https://claude.ai/code/routines) の作成画面に貼り付けて使う。

| Routine | ファイル | トリガー | 拾う対象 | 終了時のラベル |
| --- | --- | --- | --- | --- |
| 実装 | [implement.md](implement.md) | スケジュール：毎日 02:00（Asia/Tokyo） | `agent:ready` の Issue、1 回 1 件 | Issue から `agent:ready` / `agent:working` を外す。PR に `agent:review` |
| レビュー | [review.md](review.md) | GitHub：`pull_request.labeled`（Labels に `agent:review` を含む） | `agent:review` の PR | `agent:approved` か `agent:changes` |
| 修正・コンフリクト解消 | [fix.md](fix.md) | API（Actions の `dispatch-fix` / `conflict-check` / `merge-gate` から呼ぶ） | `agent:changes` か `conflict` の PR、または rebase 依頼 | `agent:review` に戻す。3 回失敗で `needs-human` |
| 定期巡回 | [patrol.md](patrol.md) | スケジュール：毎日 09:00 と 21:00（Asia/Tokyo） | 24 時間以上動きのない Issue/PR、外し忘れた `agent:working` | 状況に応じて付け直す |

## 共通の前提

- Routine の操作（コミット、PR、コメント、ラベル）は**オーナー本人の GitHub ユーザー名義**になる。author では bot と人間を区別できないので、Routine が書くコメントには必ず HTML コメントのマーカー（`<!-- agent:... -->`）を入れ、マーカーで自分の書き込みを見分ける。
- Routine はブランチ `claude/` から始まる名前で push する。
- 各プロンプトの末尾には、[共通ルール](#共通ルール各プロンプトに貼り付ける)を貼り付ける。

## 共通ルール（各プロンプトに貼り付ける）

```text
## 共通ルール（必ず守ること）
- リポジトリの CLAUDE.md を最初に読み、その規約・完了の定義・安全ルールに従う。
- main へ直接 push しない。force push は自分の claude/ ブランチに対する --force-with-lease だけ認める。
- 作業を始める前に、対象に agent:working が付いていないか確認する。付いていれば何もせずに終了する。
- 作業を始めるときに agent:working を付け、終了時は成功・失敗を問わず必ず外す。
- needs-human が付いた Issue/PR には触らない。
- 依存の追加（go.mod の require 追加）、.github/ 配下の変更、リポジトリ設定の変更、ファイルの大量削除が必要になったら、作業を止めて needs-human を付け、理由をコメントする。
- 仕様があいまいなら推測で実装しない。質問をコメントして needs-human を付ける。
- 自分が書くコメントには、指定された HTML コメントのマーカーを必ず入れる。
- 本番の認証情報・秘密情報を扱わない。コードやコメントに書かない。
- 同じ処理を繰り返していると気づいたら、needs-human を付けて止める。
```

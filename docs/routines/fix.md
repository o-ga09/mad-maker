# 修正・コンフリクト解消 Routine

## 設定

| 項目 | 値 |
| --- | --- |
| 名前 | `mad-maker: fix` |
| リポジトリ | `o-ga09/mad-maker` |
| トリガー | API のみ。作成直後に表示されるエンドポイント URL とトークンを、GitHub Secrets の `ROUTINE_FIX_URL` と `ROUTINE_FIX_TOKEN` に保存する（トークンは一度しか表示されない） |

呼び出し元（すべて GitHub Actions）:

| ワークフロー | 契機 | `reason` |
| --- | --- | --- |
| `dispatch-fix.yml` | PR に `agent:changes` が付いた | `changes-requested` |
| `conflict-check.yml` | main への push 後、PR が `dirty` になった | `conflict` |
| `merge-gate.yml` | 承認済みの PR が main より遅れている | `rebase` |

API の `text` には `repository` / `pull_request` / `reason` の 3 行が入る。

## プロンプト

```text
あなたは mad-maker リポジトリの修正担当エージェントです。API で渡されたテキストの pull_request と reason に従い、PR を 1 件修正します。

## 手順
1. 渡されたテキストから PR 番号と reason（changes-requested / conflict / rebase）を読み取る。読み取れなければ何もせずに終了する。
2. PR の状態を確認する:
   gh pr view <番号> --json number,state,labels,headRefName,headRefOid
   次のどれかに当たれば何もせずに終了する:
   - PR が open でない
   - agent:working または needs-human が付いている
   - head ブランチが claude/ で始まらない（人間のブランチには触らない）
3. 試行回数を数える: PR のコメントのうち、マーカー <!-- agent:fix-attempt --> を含むものの数。
   3 以上なら、状況をコメントして needs-human を付けて終了する。
4. PR に agent:working を付け、マーカー <!-- agent:fix-attempt --> と reason を含むコメントを残す。
5. head ブランチをチェックアウトし、reason に応じて対応する:
   - changes-requested: 最新のレビュー（マーカー <!-- agent:review ... --> のサマリとインラインコメント）の修正必須の指摘にすべて対応する。
   - conflict / rebase: git fetch origin main して origin/main へ rebase する。
     コンフリクトは両方の変更の意図を保って解消する。意図が判断できなければ rebase を中止し、
     状況をコメントして needs-human を付ける。
6. go build ./... 、go test -race ./... 、golangci-lint run がすべて通ることを確認する。
7. push する（rebase した場合は git push --force-with-lease）。
8. ラベルを付け替える: agent:changes・conflict・agent:approved を外し、agent:review を付ける。
   （rebase した場合も再レビューを経てからゲートに進む）
9. agent:working を外す。

途中で失敗した場合も、手順 9 の agent:working の除去は必ず行う。

<ここに docs/routines/README.md の「共通ルール」を貼り付ける>
```

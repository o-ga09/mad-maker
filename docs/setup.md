# ソフトウェアファクトリーの初期設定

`o-ga09/mad-maker` で、AI エージェントが「Issue → 実装 → レビュー → マージ」を回すための初期設定手順。
リポジトリは**公開**なので、GitHub Free でもブランチ保護（ruleset）と auto-merge が使える。

> [!WARNING]
> GitHub Free では、リポジトリを非公開にすると ruleset と auto-merge が**警告なしに無効になる**。非公開に切り替える場合は、マージゲートの運用を見直すこと。

## 全体像

```
人間: Issue を書いて agent:ready を付ける
  ↓ 毎日 02:00
実装 Routine ──PR (agent:review)──→ レビュー Routine ──agent:approved──→ merge-gate（Actions）──→ ready-to-merge コメント → 人間がマージ
                                       │                                    │
                                       └─agent:changes─→ dispatch-fix ─┐    └─main より遅れている─┐
main への push → conflict-check ─conflict─────────────────────────────┴──→ 修正 Routine（API）←───┘
                                                                              └─agent:review に戻す → 再レビュー
定期巡回 Routine（09:00 / 21:00）: 止まっているものを点検・再キュー
```

| 仕組み | 実体 |
| --- | --- |
| Routine（4 本） | `docs/routines/` のプロンプトを claude.ai/code/routines に登録 |
| Actions（4 本） | `ci.yml` / `conflict-check.yml` / `dispatch-fix.yml` / `merge-gate.yml` |
| 状態管理 | ラベル（`scripts/setup-labels.sh`） |

## 手順

### 1. ラベルを作る

```sh
scripts/setup-labels.sh o-ga09/mad-maker
```

何度実行しても問題ない（既存のラベルは色と説明を上書きするだけ）。

### 2. リポジトリの設定（Settings → General）

- Pull Requests
  - [x] Allow squash merging（他のマージ方法はオフ）
  - [x] Allow auto-merge
  - [x] Automatically delete head branches

### 3. ruleset を作る（Settings → Rules → Rulesets → New branch ruleset）

| 項目 | 設定 |
| --- | --- |
| Ruleset name | `protect-main` |
| Enforcement status | Active |
| Bypass list | **空のまま**（Routine はオーナー本人の名義で動くので、自分を bypass に入れると Routine も保護をすり抜ける） |
| Target branches | Include default branch |
| Restrict deletions | ON |
| Block force pushes | ON |
| Require a pull request before merging | ON。Required approvals は **0**（後述）。Allowed merge methods は Squash のみ |
| Require status checks to pass | ON。必須チェックに `ci` を追加。「Require branches to be up to date」は OFF（merge-gate の条件 4 で代替する） |

**承認数を 0 にする理由**：Routine のコミット・PR・レビューはすべてオーナー本人の GitHub ユーザー名義になる。GitHub では作成者が自分の PR を承認できないので、AI レビュー役の承認は `agent:approved` ラベルで表し、merge-gate が判定する。

### 4. Claude GitHub App をインストールする

レビュー Routine の GitHub トリガーに必要。claude.ai/code の設定から GitHub を接続し、`o-ga09/mad-maker` へのアクセスを許可する。

### 5. Routine を作る

[claude.ai/code/routines](https://claude.ai/code/routines) で、`docs/routines/` の 4 本を作る。各ファイルの「設定」表のとおりにトリガーを設定し、「プロンプト」節を貼り付ける。

- CLI の `/schedule` で作れるのはスケジュールトリガーだけ。API と GitHub のトリガーは Web 画面で追加する。
- 修正 Routine の API トリガーを作ると、エンドポイント URL とトークンが表示される。**トークンはこのときしか表示されない**ので、すぐに次の手順で Secrets に保存する。
- Routines はリサーチプレビュー中のため、画面・API・制限が変わることがある。

### 6. Secrets と Variables（Settings → Secrets and variables → Actions）

| 種類 | 名前 | 値 |
| --- | --- | --- |
| Secret | `ROUTINE_FIX_URL` | 修正 Routine の `/fire` エンドポイント URL |
| Secret | `ROUTINE_FIX_TOKEN` | 修正 Routine の API トークン |
| Variable | `AUTO_MERGE_ENABLED` | `false`（自動マージへ移行するときに `true`） |
| Variable | `REQUIRED_CHECKS` | `ci`（必須チェックを増やしたらカンマ区切りで追加） |

Secrets が未設定の間、ワークフローは Routine を起動せずに警告だけ出す。

### 7. セキュリティ機能（Settings → Advanced Security）

公開リポジトリなので、次の機能は GitHub Free でも無料で使える。

```sh
gh api -X PATCH repos/o-ga09/mad-maker --input - <<'JSON'
{"security_and_analysis":{"secret_scanning":{"status":"enabled"},"secret_scanning_push_protection":{"status":"enabled"}}}
JSON
gh api -X PUT repos/o-ga09/mad-maker/vulnerability-alerts
```

| 機能 | 役割 |
| --- | --- |
| Secret scanning + push protection | シークレットを含む push を GitHub 側で拒否する（CI の TruffleHog より前の防御） |
| Dependabot alerts | 依存の脆弱性を通知する |
| Dependabot version updates（`.github/dependabot.yml`） | コミット SHA で固定した Actions と Go の依存を週 1 回更新する PR を作る |
| Code scanning（`security.yml` の `codeql`） | CodeQL の結果を Security タブに表示する |

**Action の固定ルール**：ワークフローの `uses:` はすべてコミット SHA で固定し、行末にバージョンをコメントで書く（例: `actions/checkout@<40桁のSHA> # v7.0.1`）。Dependabot はこの形式のまま SHA とコメントを更新する。Dependabot の PR は `.github/` や `go.mod` を変更するので、人間が確認してマージする。

### 8. 動作確認

1. `.github/ISSUE_TEMPLATE/agent-task.md` のテンプレートで、小さな Issue を 1 件作り、`agent:ready` を付ける。
2. 実装 Routine を Web 画面の **Run now** で実行する。
3. 次の点を確認し、結果をこのファイルに追記する。
   - PR の作成者とコミットの author が誰になっているか（ドキュメント上はオーナー本人の名義）
   - レビュー Routine が `agent:review` の付与で起動するか
   - `agent:changes` を付けると `dispatch-fix` が修正 Routine を起動するか
   - `agent:approved` かつ CI 成功で、merge-gate が `ready-to-merge` をコメントするか

## 運用

- 最初の数週間は、Routine の実行履歴（claude.ai/code/routines の History）と利用量を毎日確認する。ループしていたら、該当 Routine を一時停止する。
- `needs-human` が付いたものを見て判断する。判断し終えたら `needs-human` を外し、必要に応じて `agent:ready` / `agent:review` を付け直す。
- レビューで繰り返し出る指摘や失敗パターンは、CLAUDE.md のチェックリスト・規約に反映する。

### 自動マージへの移行基準

**連続 5 件の PR で、人間の修正なしにマージできたら**自動マージへ移行する。

1. Variable `AUTO_MERGE_ENABLED` を `true` にする。
2. merge-gate は、条件を満たした PR に `gh pr merge --squash --auto` を設定する。マージは ruleset の条件（CI 成功）を満たした時点で GitHub が行う。

注意：`GITHUB_TOKEN` によるマージでは、他のワークフローが起動しない。そのため自動マージ後の main への push では `conflict-check` が動かない。移行時は、`conflict-check.yml` にスケジュール実行を追加するか、定期巡回でコンフリクトを点検する。

## 将来の拡張

- ローカル資源・GPU が必要になったら、自宅サーバ / VPS の self-hosted runner を足す（実装は Web、CI・デプロイは自前のハイブリッド構成）。
- Actions の無料枠が窮屈になったら、self-hosted runner か GitHub Pro を検討する。
- GitHub のレビュー機能で承認を扱いたくなったら、bot アカウントか GitHub App を用意する（その場合も認証情報を Routine に直接渡さない設計にする）。

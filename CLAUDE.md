# CLAUDE.md

このリポジトリは、AIエージェント（Claude Code on the web の Routines）が「Issue → 実装 → レビュー → マージ」を回すソフトウェアファクトリーとして運用する。
人間（オーナー）の役割は、Issue（仕様）を書くことと、`needs-human` が付いたものを判断すること。

## プロジェクト概要

- mad-maker：画像と音楽を入れると、ビートに合わせて切り替わる MAD 動画を作るアプリ
- MVP：Go + Ebitengine（または guigui）のデスクトップアプリ
- 次の段階：WebAssembly でブラウザ版
- ライセンス：All Rights Reserved（外部コードの貼り付け・ライセンス不明なアセットの追加は禁止）

## コマンド

| 目的 | コマンド |
| --- | --- |
| ビルド | `go build ./...` |
| テスト | `go test -race ./...`（Linux CI では `xvfb-run` 経由） |
| lint | `golangci-lint run` |
| フォーマット | `gofmt -w .`（golangci-lint の formatter でも検査される） |
| ラベル作成 | `scripts/setup-labels.sh` |

PR を出す前に、ビルド・テスト・lint をローカルで実行し、すべて通っていること。

## コーディング規約

- Go の標準的なスタイル（Effective Go / Go Code Review Comments）に従う。
- エラーは握りつぶさない。`fmt.Errorf("...: %w", err)` で文脈を付けて返す。
- 描画・音声などの Ebitengine 依存は薄い層に閉じ込め、ビート検出・タイムライン計算などのロジックは Ebitengine 抜きでテストできるようにする。
- テストはテーブル駆動を基本とする。振る舞いを追加・変更するときは、先に失敗するテストを書く（TDD）。
- 1 PR = 1 Issue。Issue の「対象範囲」に書かれていないファイルは変更しない。

## 完了の定義（Definition of Done）

1. Issue の完了条件をすべて満たしている。
2. 追加・変更した振る舞いにテストがある。
3. `go build ./...`、`go test -race ./...`、`golangci-lint run` が通る。
4. PR 本文が `.github/PULL_REQUEST_TEMPLATE.md` に沿って書かれている（`Closes #<Issue番号>`、変更内容、テスト方法、チェックリスト）。
5. 「人間の承認が必要な操作」（後述）に当たる変更が含まれていない。含まれる場合は `needs-human` を付けている。

## レビューのチェックリスト

レビュー Routine はこの順で確認し、問題があれば該当行にインラインコメントを付ける。

- [ ] Issue の完了条件を満たしているか。範囲外の変更が混ざっていないか
- [ ] テストが振る舞いを検証しているか（実装の写しになっていないか）。境界値・エラー系があるか
- [ ] エラー処理が適切か（握りつぶし・panic の乱用がないか）
- [ ] goroutine・チャネルのリークや競合がないか
- [ ] 毎フレーム実行される処理（`Update` / `Draw`）で不要なメモリ確保・重い計算をしていないか
- [ ] 秘密情報・個人情報・絶対パスがコードやログに入っていないか
- [ ] ライセンス不明なコード・アセットが入っていないか
- [ ] 依存の追加、CI 設定・リポジトリ設定の変更、大量削除がないか（あれば `needs-human`）
- [ ] 命名・コメントがわかりやすく、不要なコードが残っていないか

## 作業状態のラベル

| ラベル | 付く対象 | 意味 | 付ける主体 |
| --- | --- | --- | --- |
| `agent:ready` | Issue | 実装してよい。仕様と完了条件が揃っている | 人間 |
| `agent:working` | Issue / PR | エージェントが作業中。他の Routine は触らない（排他ロック） | Routine（開始時に付け、終了時に外す） |
| `agent:review` | PR | レビュー待ち | 実装 Routine・修正 Routine |
| `agent:changes` | PR | レビューで修正依頼あり | レビュー Routine |
| `agent:approved` | PR | レビュー役が承認した | レビュー Routine |
| `conflict` | PR | main とコンフリクトしている | conflict-check（Actions） |
| `needs-human` | Issue / PR | 自動処理を止めて人間の判断を待つ | 全 Routine |

ラベルは誰でも付け外しできるので、マージしてよいかの最終判定には使わない。最終判定は `merge-gate` ワークフローと人間が行う。

## 安全ルール（必ず守ること）

- **ループ防止**：bot 自身の push・コメント・ラベル変更には反応しない。同じ PR への自動修正は最大 3 回まで。超えたら `needs-human` を付けて止める。
- **排他**：作業を始めるときに `agent:working` を付け、すでに付いているものには触らない。終了時には、失敗した場合も含めて必ず外す。
- **main への直接 push 禁止**：すべての変更は PR 経由にする。force push もしない。
- **秘密情報**：本番の認証情報を Routine の環境に渡さない。必要なものは GitHub Secrets に置き、Actions の中だけで使う。
- **対象の絞り込み**：`agent:ready` が付いた Issue だけを扱う。並列数は 1。
- **コスト**：ループすると利用枠が一気に減る。同じ処理を繰り返していると気づいたら、`needs-human` を付けて止める。
- **人間の承認が必要な操作**：依存関係の追加（`go.mod` の require 追加）、CI 設定の変更（`.github/` 配下）、リポジトリ設定の変更、ファイルの大量削除。これらを含む PR には `needs-human` を付ける。
- **判断に迷ったら止まる**：仕様があいまいな場合、推測で実装しない。Issue に質問をコメントし、`needs-human` を付ける。

## ブランチとコミット

- ブランチ名：`agent/<Issue番号>-<短い説明>`（人間の作業は `feat/...`・`fix/...`・`chore/...`）
- コミットメッセージ：Conventional Commits（`feat:` `fix:` `test:` `refactor:` `docs:` `chore:` `ci:`）
- マージ方法：squash merge

## 関連ドキュメント

- `docs/setup.md`：ファクトリーの初期設定手順（ラベル、ruleset、Routines、Secrets）
- `docs/routines/`：各 Routine のプロンプト

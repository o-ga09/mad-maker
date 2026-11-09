#!/usr/bin/env bash
# ソフトウェアファクトリーで使う作業状態ラベルを作成する。
# 既存のラベルは --force で色・説明を上書きするだけなので、何度実行しても壊れない。
#
# 使い方: scripts/setup-labels.sh [owner/repo]
#   引数を省略すると、カレントディレクトリのリポジトリを対象にする。
set -euo pipefail

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh (GitHub CLI) が見つかりません" >&2
  exit 1
fi

repo_args=()
if [[ $# -ge 1 ]]; then
  repo_args=(--repo "$1")
fi

# name|color|description
labels=(
  "agent:ready|0e8a16|実装してよい。仕様と完了条件が揃っている（人間が付ける）"
  "agent:working|fbca04|エージェントが作業中。他のRoutineは触らない（排他ロック）"
  "agent:review|1d76db|レビュー待ち"
  "agent:changes|d93f0b|レビューで修正依頼あり"
  "agent:approved|0e8a16|レビュー役が承認した"
  "conflict|b60205|mainとコンフリクトしている"
  "needs-human|5319e7|自動処理を止めて人間の判断を待つ"
)

for entry in "${labels[@]}"; do
  IFS='|' read -r name color description <<<"$entry"
  gh label create "$name" --color "$color" --description "$description" --force "${repo_args[@]}"
  echo "ok: $name"
done

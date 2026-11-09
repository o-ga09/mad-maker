#!/usr/bin/env bash
# マージゲート: PR が次の 4 条件をすべて満たすか判定する。
#   1. 必須の CI ジョブがすべて成功している
#   2. agent:approved が付いていて、needs-human・conflict・agent:working が付いていない
#   3. mergeable_state が clean である（unknown の間は待って取り直す）
#   4. head ブランチが最新の main を含んでいる（含んでいなければ修正 Routine に rebase を依頼する）
#
# すべて満たした場合:
#   AUTO_MERGE_ENABLED=true  → gh pr merge --squash --auto
#   それ以外（既定）         → ready-to-merge コメントを残すだけ（head SHA ごとに 1 回）
#
# 必要な環境変数:
#   GH_TOKEN, REPO, PR_NUMBER
#   REQUIRED_CHECKS     必須チェック名（カンマ区切り、既定: ci）
#   AUTO_MERGE_ENABLED  "true" で自動マージを有効にする（既定: 無効）
#   ROUTINE_URL, ROUTINE_TOKEN  修正 Routine の API（rebase 依頼用、未設定なら依頼しない）
set -euo pipefail

: "${REPO:?}" "${PR_NUMBER:?}"
REQUIRED_CHECKS="${REQUIRED_CHECKS:-ci}"
AUTO_MERGE_ENABLED="${AUTO_MERGE_ENABLED:-false}"
script_dir="$(cd "$(dirname "$0")" && pwd)"

summary() { echo "$1" | tee -a "${GITHUB_STEP_SUMMARY:-/dev/null}"; }

pr_json="$(gh api "repos/$REPO/pulls/$PR_NUMBER")"
state="$(jq -r '.state' <<<"$pr_json")"
draft="$(jq -r '.draft' <<<"$pr_json")"
head_sha="$(jq -r '.head.sha' <<<"$pr_json")"
head_repo="$(jq -r '.head.repo.full_name' <<<"$pr_json")"
base_ref="$(jq -r '.base.ref' <<<"$pr_json")"
labels=",$(jq -r '[.labels[].name] | join(",")' <<<"$pr_json"),"

summary "## merge-gate: PR #$PR_NUMBER (${head_sha:0:7})"

if [[ "$state" != "open" || "$draft" == "true" || "$head_repo" != "$REPO" || "$base_ref" != "main" ]]; then
  summary "対象外（closed / draft / fork / base が main 以外）"
  exit 0
fi

ok=true

# 条件 1: 必須 CI
checks_json="$(gh api "repos/$REPO/commits/$head_sha/check-runs?per_page=100")"
IFS=',' read -ra required <<<"$REQUIRED_CHECKS"
for name in "${required[@]}"; do
  conclusion="$(jq -r --arg n "$name" '[.check_runs[] | select(.name == $n)] | sort_by(.completed_at) | last | .conclusion // "pending"' <<<"$checks_json")"
  if [[ "$conclusion" == "success" ]]; then
    summary "- ✅ CI \`$name\`: success"
  else
    summary "- ❌ CI \`$name\`: $conclusion"
    ok=false
  fi
done

# 条件 2: ラベル
if [[ "$labels" == *",agent:approved,"* ]]; then
  summary "- ✅ \`agent:approved\` あり"
else
  summary "- ❌ \`agent:approved\` なし"
  ok=false
fi
for blocker in needs-human conflict agent:working; do
  if [[ "$labels" == *",$blocker,"* ]]; then
    summary "- ❌ \`$blocker\` が付いている"
    ok=false
  fi
done

# 条件 3: mergeable_state
mergeable_state="unknown"
for _ in 1 2 3 4 5 6; do
  mergeable_state="$(gh api "repos/$REPO/pulls/$PR_NUMBER" --jq '.mergeable_state')"
  [[ "$mergeable_state" != "unknown" ]] && break
  sleep 10
done
if [[ "$mergeable_state" == "clean" ]]; then
  summary "- ✅ mergeable_state: clean"
else
  summary "- ❌ mergeable_state: $mergeable_state"
  ok=false
fi

# 条件 4: 最新の main を含んでいるか
behind_by="$(gh api "repos/$REPO/compare/main...$head_sha" --jq '.behind_by')"
up_to_date=true
if [[ "$behind_by" == "0" ]]; then
  summary "- ✅ 最新の main を含んでいる"
else
  summary "- ❌ main より $behind_by コミット遅れている"
  ok=false
  up_to_date=false
fi

# コメント済みかどうかを head SHA ごとのマーカーで判定する（同じ内容を何度も書かない）
has_marker() {
  gh api "repos/$REPO/issues/$PR_NUMBER/comments?per_page=100" --paginate \
    --jq ".[].body | select(contains(\"$1\"))" | grep -q .
}

if [[ "$ok" == "true" ]]; then
  if [[ "$AUTO_MERGE_ENABLED" == "true" ]]; then
    summary "→ すべての条件を満たしたため、自動マージを設定します"
    gh pr merge "$PR_NUMBER" --repo "$REPO" --squash --auto --match-head-commit "$head_sha"
  else
    marker="<!-- merge-gate:ready ${head_sha} -->"
    if has_marker "$marker"; then
      summary "→ ready-to-merge は通知済み"
    else
      gh pr comment "$PR_NUMBER" --repo "$REPO" --body "ready-to-merge ✅ マージゲートの全条件を満たしました（${head_sha:0:7}）。人間がマージしてください。
$marker"
      summary "→ ready-to-merge をコメントしました"
    fi
  fi
  exit 0
fi

# 条件 4 だけが原因で止まっている場合は、修正 Routine に rebase を依頼する（head SHA ごとに 1 回）
if [[ "$up_to_date" == "false" && "$labels" == *",agent:approved,"* \
      && "$labels" != *",needs-human,"* && "$labels" != *",agent:working,"* && "$labels" != *",conflict,"* ]]; then
  marker="<!-- merge-gate:rebase-requested ${head_sha} -->"
  if has_marker "$marker"; then
    summary "→ rebase は依頼済み"
  else
    gh pr comment "$PR_NUMBER" --repo "$REPO" --body "merge-gate: main より遅れているため、修正 Routine に rebase を依頼しました。
$marker"
    "$script_dir/fire-routine.sh" "repository: $REPO
pull_request: #$PR_NUMBER
reason: rebase
PR #$PR_NUMBER は承認済みですが main より遅れています。main へ rebase して push してください。"
  fi
fi

summary "→ 条件を満たしていないため待機します"

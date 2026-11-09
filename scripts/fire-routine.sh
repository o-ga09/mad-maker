#!/usr/bin/env bash
# Claude Code の Routine（scheduled task）を API トリガーで起動する。
# https://code.claude.com/docs/en/web-scheduled-tasks.md の「Trigger a scheduled task」に準拠。
#
# 必要な環境変数:
#   ROUTINE_URL    Routine の /fire エンドポイント（例: https://api.anthropic.com/v1/claude_code/routines/trig_xxx/fire）
#   ROUTINE_TOKEN  Routine の API トリガートークン（GitHub Secrets から渡す）
# 引数:
#   $1  Routine に渡す run 固有のコンテキスト（自由形式テキスト）
#
# ROUTINE_URL / ROUTINE_TOKEN が未設定の場合は警告だけ出して成功終了する（初期セットアップ中に CI を落とさないため）。
set -euo pipefail

text="${1:?usage: fire-routine.sh <text>}"

if [[ -z "${ROUTINE_URL:-}" || -z "${ROUTINE_TOKEN:-}" ]]; then
  echo "::warning::ROUTINE_URL / ROUTINE_TOKEN が未設定のため Routine を起動しませんでした"
  exit 0
fi

body="$(jq -n --arg text "$text" '{text: $text}')"

status="$(curl -sS -o /tmp/fire-routine-response.json -w '%{http_code}' \
  -X POST "$ROUTINE_URL" \
  -H "Authorization: Bearer ${ROUTINE_TOKEN}" \
  -H "anthropic-beta: experimental-cc-routine-2026-04-01" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d "$body")"

if [[ "$status" -lt 200 || "$status" -ge 300 ]]; then
  echo "::error::Routine の起動に失敗しました (HTTP $status)"
  cat /tmp/fire-routine-response.json >&2 || true
  exit 1
fi

echo "Routine を起動しました (HTTP $status)"

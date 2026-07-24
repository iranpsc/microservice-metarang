#!/usr/bin/env bash
# Run govulncheck and append a markdown checklist entry to GITHUB_STEP_SUMMARY.
#
# Usage:
#   govulncheck-checklist.sh <section_title> <workdir>

set -uo pipefail

TITLE="${1:?section title required}"
WORKDIR="${2:?workdir required}"
SUMMARY="${GITHUB_STEP_SUMMARY:-/dev/stdout}"
LOG_FILE="$(mktemp)"
trap 'rm -f "$LOG_FILE"' EXIT

cd "$WORKDIR"

{
  echo ""
  echo "### ${TITLE}"
  echo ""
} >> "$SUMMARY"

set +e
govulncheck ./... > "$LOG_FILE" 2>&1
EXIT_CODE=$?
set -e

if [ "$EXIT_CODE" -eq 0 ]; then
  echo "- [x] No known vulnerabilities found" >> "$SUMMARY"
  echo "--- ${TITLE} ---"
  cat "$LOG_FILE"
  exit 0
fi

echo "- [ ] **Vulnerabilities detected** ❌ FAILED" >> "$SUMMARY"
echo '```' >> "$SUMMARY"
cat "$LOG_FILE" >> "$SUMMARY"
echo '```' >> "$SUMMARY"

echo "--- ${TITLE} ---"
cat "$LOG_FILE"
exit "$EXIT_CODE"

#!/usr/bin/env bash
# Run govulncheck and append a markdown checklist entry to GITHUB_STEP_SUMMARY.
# Job logs stay compact on success; full output only on failure.
#
# Usage:
#   govulncheck-checklist.sh <section_title> <workdir>

set -uo pipefail

TITLE="${1:?section title required}"
WORKDIR="${2:?workdir required}"
SUMMARY="${GITHUB_STEP_SUMMARY:-/dev/null}"
LOG_FILE="$(mktemp)"
trap 'rm -f "$LOG_FILE"' EXIT

cd "$WORKDIR"

{
  echo ""
  echo "### ${TITLE}"
  echo ""
} >> "$SUMMARY"

echo "::group::${TITLE}"

set +e
govulncheck ./... >"$LOG_FILE" 2>&1
EXIT_CODE=$?
set -e

if [ "$EXIT_CODE" -eq 0 ]; then
  echo "- [x] No known vulnerabilities found" >> "$SUMMARY"
  echo "✓ No known vulnerabilities found"
  echo "::endgroup::"
  exit 0
fi

{
  echo "- [ ] **Vulnerabilities detected** ❌ FAILED"
  echo ""
  echo '```'
  # Cap summary size to stay within GitHub's step-summary limits.
  head -n 200 "$LOG_FILE"
  echo '```'
} >> "$SUMMARY"

echo "::error::govulncheck found vulnerabilities"
# Cap job-log dump as well.
head -n 200 "$LOG_FILE"
echo "::endgroup::"
exit "$EXIT_CODE"

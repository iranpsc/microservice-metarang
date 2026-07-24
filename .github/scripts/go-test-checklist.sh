#!/usr/bin/env bash
# Run go test with -json output and append a markdown checklist to GITHUB_STEP_SUMMARY.
#
# Usage:
#   go-test-checklist.sh <section_title> <workdir> -- [go test args...]
#
# Example:
#   go-test-checklist.sh "Unit Tests" "services/auth-service" -- ./internal/... -race

set -uo pipefail

TITLE="${1:?section title required}"
WORKDIR="${2:?workdir required}"
shift 2

if [ "${1:-}" = "--" ]; then
  shift
fi

TEST_ARGS=("$@")
SUMMARY="${GITHUB_STEP_SUMMARY:-/dev/stdout}"
JSON_LOG="$(mktemp)"
trap 'rm -f "$JSON_LOG"' EXIT

cd "$WORKDIR"

{
  echo ""
  echo "### ${TITLE}"
  echo ""
} >> "$SUMMARY"

set +e
go test "${TEST_ARGS[@]}" -json > "$JSON_LOG" 2>&1
EXIT_CODE=$?
set -e

if ! command -v jq >/dev/null 2>&1; then
  if [ "$EXIT_CODE" -eq 0 ]; then
    echo "✅ All tests passed" >> "$SUMMARY"
  else
    echo "❌ Tests failed (exit code: ${EXIT_CODE})" >> "$SUMMARY"
    echo '```' >> "$SUMMARY"
    tail -50 "$JSON_LOG" >> "$SUMMARY"
    echo '```' >> "$SUMMARY"
  fi
  exit "$EXIT_CODE"
fi

FIRST_LINE="$(head -n 1 "$JSON_LOG" || true)"
if [ -n "$FIRST_LINE" ] && ! echo "$FIRST_LINE" | jq -e . >/dev/null 2>&1; then
  echo "❌ **Test run failed**" >> "$SUMMARY"
  echo '```' >> "$SUMMARY"
  cat "$JSON_LOG" >> "$SUMMARY"
  echo '```' >> "$SUMMARY"
  echo "--- ${TITLE} ---"
  cat "$JSON_LOG"
  exit "$EXIT_CODE"
fi

PACKAGE_FAILURES="$(jq -rs '
  [.[] | select(.Action == "fail" and (.Test == null or .Test == ""))]
  | unique_by(.Package)
  | .[]
  | "- [ ] **" + .Package + "** ❌ FAILED"
' "$JSON_LOG" 2>/dev/null || true)"

if [ -n "$PACKAGE_FAILURES" ]; then
  {
    echo "**Package failures:**"
    echo "$PACKAGE_FAILURES"
    echo ""
  } >> "$SUMMARY"
fi

PACKAGE_PASSES="$(jq -rs '
  ( [.[] | select(.Test != null and .Test != "")] | map(.Package) | unique ) as $with_tests |
  [.[] | select(.Action == "pass" and (.Test == null or .Test == ""))]
  | unique_by(.Package)
  | map(select(.Package as $p | ($with_tests | index($p)) | not))
  | .[]
  | "- [x] **" + .Package + "** (no test functions)"
' "$JSON_LOG" 2>/dev/null || true)"

if [ -n "$PACKAGE_PASSES" ]; then
  {
    echo "**Packages without test functions:**"
    echo "$PACKAGE_PASSES"
    echo ""
  } >> "$SUMMARY"
fi

jq -rs '
  [.[] | select(.Action == "pass" or .Action == "fail" or .Action == "skip") | select(.Test != null and .Test != "")]
  | group_by(.Package)
  | sort_by(.[0].Package)
  | .[]
  | "#### `" + .[0].Package + "`\n" +
    (map(
      if .Action == "pass" then "- [x] `" + .Test + "`"
      elif .Action == "fail" then "- [ ] `" + .Test + "` ❌ **FAILED**"
      else "- [ ] `" + .Test + "` ⏭️ skipped"
      end
    ) | unique | join("\n"))
' "$JSON_LOG" >> "$SUMMARY" 2>/dev/null || true

jq -rs '
  ( [.[] | select(.Action == "pass" and .Test != null and .Test != "")] | length ) as $passed |
  ( [.[] | select(.Action == "fail" and .Test != null and .Test != "")] | length ) as $failed |
  ( [.[] | select(.Action == "skip" and .Test != null and .Test != "")] | length ) as $skipped |
  ( [.[] | select(.Action == "fail" and (.Test == null or .Test == ""))] | length ) as $pkg_failed |
  "\n| Status | Count |\n|--------|-------|\n| ✅ Passed | \($passed) |\n| ❌ Failed | \($failed) |\n| ⏭️ Skipped | \($skipped) |\n| 📦 Package failures | \($pkg_failed) |"
' "$JSON_LOG" >> "$SUMMARY" 2>/dev/null || true

echo "--- ${TITLE} ---"
FAILED_TESTS="$(jq -rs '
  [.[] | select(.Action == "fail")]
  | .[]
  | if .Test then "\(.Package) \(.Test): FAILED" else "\(.Package): PACKAGE FAILED" end
' "$JSON_LOG" 2>/dev/null || true)"

if [ -n "$FAILED_TESTS" ]; then
  echo "$FAILED_TESTS"
fi

PASSED="$(jq -rs '[.[] | select(.Action == "pass" and .Test != null and .Test != "")] | length' "$JSON_LOG" 2>/dev/null || echo 0)"
FAILED="$(jq -rs '[.[] | select(.Action == "fail" and .Test != null and .Test != "")] | length' "$JSON_LOG" 2>/dev/null || echo 0)"
echo "Summary: ${PASSED} passed, ${FAILED} failed"

exit "$EXIT_CODE"

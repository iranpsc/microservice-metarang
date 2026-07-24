#!/usr/bin/env bash
# Run go test with -json output and append a markdown checklist to GITHUB_STEP_SUMMARY.
# Job logs stay compact: package-level progress plus failure details only.
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
SUMMARY="${GITHUB_STEP_SUMMARY:-/dev/null}"
JSON_LOG="$(mktemp)"
ERR_LOG="$(mktemp)"
trap 'rm -f "$JSON_LOG" "$ERR_LOG"' EXIT

cd "$WORKDIR"

{
  echo ""
  echo "### ${TITLE}"
  echo ""
} >> "$SUMMARY"

echo "::group::${TITLE}"

set +e
# Keep stdout as NDJSON only; send compiler/tool noise elsewhere.
go test "${TEST_ARGS[@]}" -json >"$JSON_LOG" 2>"$ERR_LOG"
EXIT_CODE=$?
set -e

if [ -s "$ERR_LOG" ]; then
  echo "::warning::go test wrote to stderr (first 30 lines):"
  head -n 30 "$ERR_LOG"
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "::error::jq is required to render the test checklist"
  echo "❌ Tests finished but jq is unavailable (exit code: ${EXIT_CODE})" >> "$SUMMARY"
  echo "::endgroup::"
  exit "$EXIT_CODE"
fi

# Detect non-JSON stdout (build failures often print plain text).
if [ -s "$JSON_LOG" ] && ! head -n 1 "$JSON_LOG" | jq -e . >/dev/null 2>&1; then
  {
    echo "❌ **Test run failed before producing JSON results**"
    echo ""
    echo '```'
    head -n 80 "$JSON_LOG"
    echo '```'
  } >> "$SUMMARY"
  echo "::error::go test did not produce JSON output"
  head -n 80 "$JSON_LOG"
  echo "::endgroup::"
  exit "${EXIT_CODE:-1}"
fi

# Compact console: one line per package.
jq -rs '
  [.[] | select(.Action == "pass" or .Action == "fail" or .Action == "skip")]
  | group_by(.Package)
  | sort_by(.[0].Package)
  | .[]
  | (map(select(.Test != null and .Test != "")) | length) as $tests
  | (map(select(.Action == "fail")) | length) as $fails
  | if $fails > 0 then
      "✗ \(.[0].Package) (\($fails) failed)"
    elif any(.Action == "pass" and (.Test == null or .Test == "")) or $tests > 0 then
      "✓ \(.[0].Package) (\($tests) tests)"
    else
      empty
    end
' "$JSON_LOG" 2>/dev/null || true

FAILED_KEYS="$(jq -cs '
  [.[] | select(.Action == "fail" and .Test != null and .Test != "")
   | {pkg: .Package, test: .Test}]
  | unique_by(.pkg + "/" + .test)
' "$JSON_LOG" 2>/dev/null || echo '[]')"

FAILED_COUNT="$(echo "$FAILED_KEYS" | jq 'length')"

if [ "$FAILED_COUNT" -gt 0 ]; then
  echo ""
  echo "Failed tests:"
  echo "$FAILED_KEYS" | jq -r '.[] | "FAIL\t\(.pkg)\t\(.test)"'

  echo ""
  echo "Failure output:"
  jq -rs --argjson fails "$FAILED_KEYS" '
    ($fails | map(.pkg + "\t" + .test)) as $keys
    | [.[]
       | select(.Action == "output" and .Test != null and (.Output | type == "string"))
       | select((.Package + "\t" + .Test) as $k | ($keys | index($k)) != null)
      ]
    | group_by(.Package + "/" + .Test)
    | .[]
    | "--- \(.[0].Package) \(.[0].Test) ---\n" + (map(.Output) | join(""))
  ' "$JSON_LOG" 2>/dev/null || true
fi

# Step summary: package checklist + individual failures only.
{
  echo "**Packages:**"
  echo ""
} >> "$SUMMARY"

jq -rs '
  [.[] | select(.Action == "pass" or .Action == "fail" or .Action == "skip")]
  | group_by(.Package)
  | sort_by(.[0].Package)
  | .[]
  | (map(select(.Test != null and .Test != "")) | length) as $tests
  | (map(select(.Action == "fail" and .Test != null)) | length) as $fails
  | (map(select(.Action == "skip" and .Test != null)) | length) as $skips
  | (any(.Action == "fail" and (.Test == null or .Test == ""))) as $pkg_fail
  | if $pkg_fail or $fails > 0 then
      "- [ ] `\(.[0].Package)` ❌ failed (\($fails) tests)"
    elif $tests == 0 then
      "- [x] `\(.[0].Package)` (no test functions)"
    elif $skips > 0 then
      "- [x] `\(.[0].Package)` (\($tests) tests, \($skips) skipped)"
    else
      "- [x] `\(.[0].Package)` (\($tests) tests)"
    end
' "$JSON_LOG" >> "$SUMMARY" 2>/dev/null || true

if [ "$FAILED_COUNT" -gt 0 ]; then
  {
    echo ""
    echo "**Failed tests:**"
    echo ""
    echo "$FAILED_KEYS" | jq -r '.[] | "- [ ] `\(.pkg)` · `\(.test)` ❌"'
  } >> "$SUMMARY"
fi

jq -rs '
  ( [.[] | select(.Action == "pass" and .Test != null and .Test != "")] | length ) as $passed |
  ( [.[] | select(.Action == "fail" and .Test != null and .Test != "")] | length ) as $failed |
  ( [.[] | select(.Action == "skip" and .Test != null and .Test != "")] | length ) as $skipped |
  ( [.[] | select(.Action == "fail" and (.Test == null or .Test == ""))] | unique_by(.Package) | length ) as $pkg_failed |
  "\n| Status | Count |\n|--------|-------|\n| ✅ Passed | \($passed) |\n| ❌ Failed | \($failed) |\n| ⏭️ Skipped | \($skipped) |\n| 📦 Package failures | \($pkg_failed) |"
' "$JSON_LOG" >> "$SUMMARY" 2>/dev/null || true

PASSED="$(jq -rs '[.[] | select(.Action == "pass" and .Test != null and .Test != "")] | length' "$JSON_LOG" 2>/dev/null || echo 0)"
FAILED="$(jq -rs '[.[] | select(.Action == "fail" and .Test != null and .Test != "")] | length' "$JSON_LOG" 2>/dev/null || echo 0)"
echo ""
echo "Result: ${PASSED} passed, ${FAILED} failed (exit ${EXIT_CODE})"

echo "::endgroup::"
exit "$EXIT_CODE"

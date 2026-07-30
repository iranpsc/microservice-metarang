#!/usr/bin/env bash
# Merge Go coverage profiles and append a markdown summary to GITHUB_STEP_SUMMARY.
#
# Usage:
#   go-coverage-summary.sh <section_title> <merged_output> [profile...]
#
# Example:
#   go-coverage-summary.sh "Code Coverage" coverage.out unit.out integration.out

set -euo pipefail

TITLE="${1:?section title required}"
OUTPUT="${2:?merged output path required}"
shift 2

SUMMARY="${GITHUB_STEP_SUMMARY:-/dev/null}"
PROFILES=()

for profile in "$@"; do
  if [ -f "$profile" ] && [ -s "$profile" ]; then
    PROFILES+=("$profile")
  fi
done

{
  echo ""
  echo "### ${TITLE}"
  echo ""
} >> "$SUMMARY"

if [ "${#PROFILES[@]}" -eq 0 ]; then
  {
    echo "_No coverage data was collected._"
    echo ""
    echo "Ensure tests run with \`-coverprofile\` and, for external test modules,"
    echo "use \`-coverpkg=metarang/<service>/...\` to measure service code."
  } >> "$SUMMARY"
  exit 0
fi

if [ "${#PROFILES[@]}" -eq 1 ]; then
  cp "${PROFILES[0]}" "$OUTPUT"
else
  if ! command -v gocovmerge >/dev/null 2>&1; then
    go install github.com/wadey/gocovmerge@latest
  fi
  gocovmerge "${PROFILES[@]}" > "$OUTPUT"
fi

TOTAL_LINE="$(go tool cover -func="$OUTPUT" | tail -1)"
TOTAL_PCT="$(echo "$TOTAL_LINE" | grep -oE '[0-9]+\.[0-9]+%' | tail -1 || true)"

if [ -z "$TOTAL_PCT" ]; then
  {
    echo "_Coverage profile exists but total could not be parsed._"
    echo ""
    echo '```'
    echo "$TOTAL_LINE"
    echo '```'
  } >> "$SUMMARY"
  exit 0
fi

{
  echo "**Total statement coverage: ${TOTAL_PCT}**"
  echo ""
  echo "| Package | Coverage |"
  echo "|---------|----------|"
} >> "$SUMMARY"

go tool cover -func="$OUTPUT" \
  | awk -F'\t' '
    $1 != "total:" {
      pkg = $1
      sub(/:[0-9]+:[0-9]+,.*/, "", pkg)
      sub(/\/[^/]+$/, "", pkg)
      if (pkg !~ /^metarang\//) {
        next
      }
      pct = $NF
      gsub(/%$/, "", pct)
      sum[pkg] += pct
      count[pkg]++
    }
    END {
      for (pkg in sum) {
        printf "| `%s` | %.1f%% |\n", pkg, sum[pkg] / count[pkg]
      }
    }
  ' \
  | sort \
  >> "$SUMMARY"

echo "" >> "$SUMMARY"
echo "Coverage profiles merged: ${#PROFILES[@]} file(s) → \`${OUTPUT}\`" >> "$SUMMARY"

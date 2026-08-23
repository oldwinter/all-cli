#!/usr/bin/env bash
set -euo pipefail

coverage_file="${COVERAGE_FILE:-coverage.out}"
minimum="${COVERAGE_MIN:-80.0}"

env -u GOROOT -u GOTOOLDIR go test -coverprofile="$coverage_file" ./...

total="$(
  env -u GOROOT -u GOTOOLDIR go tool cover -func="$coverage_file" |
    awk '/^total:/ {gsub(/%/, "", $3); print $3}'
)"

if [[ -z "$total" ]]; then
  echo "coverage gate: could not read total coverage from $coverage_file" >&2
  exit 1
fi

printf 'coverage gate: total=%s%% minimum=%s%%\n' "$total" "$minimum"
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo "### Coverage"
    echo
    echo "| Metric | Value |"
    echo "|---|---:|"
    echo "| Total statement coverage | ${total}% |"
    echo "| Required minimum | ${minimum}% |"
  } >>"$GITHUB_STEP_SUMMARY"
fi

if ! awk -v actual="$total" -v required="$minimum" 'BEGIN { exit !(actual + 0 >= required + 0) }'; then
  echo "coverage gate failed: ${total}% is below ${minimum}%" >&2
  exit 1
fi

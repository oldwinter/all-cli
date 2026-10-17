#!/usr/bin/env bash
set -euo pipefail

coverage_file="${COVERAGE_FILE:-coverage.out}"
minimum="${COVERAGE_MIN:-80.0}"

# Pick a `go` invocation that works on this host. `env -u` is POSIX-only, and
# under WSL/Git-Bash only `go.exe` is on PATH while native Linux needs the
# GOROOT-clearing wrapper.
go_bin="go"
if command -v go >/dev/null 2>&1; then
  go_bin="go"
elif command -v go.exe >/dev/null 2>&1; then
  go_bin="go.exe"
elif env -u GOROOT -u GOTOOLDIR go version >/dev/null 2>&1; then
  go_bin="env -u GOROOT -u GOTOOLDIR go"
else
  echo "coverage gate: no working 'go' executable found" >&2
  exit 127
fi

# -coverpkg=./... instruments every package from every test binary. Without it
# Go only counts a package's own tests, which under-reports code exercised
# across package boundaries (internal/diagnose read 49% standalone but ~79%
# when driven from internal/cli tests).
$go_bin test -coverpkg=./... -coverprofile="$coverage_file" ./...

total="$(
  $go_bin tool cover -func="$coverage_file" |
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

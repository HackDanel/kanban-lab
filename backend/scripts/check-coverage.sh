#!/bin/sh
# Fails when statement coverage in coverage.out is below MIN_LINES (default 80).
set -eu
cd "$(dirname "$0")/.."
total=$(go tool cover -func=coverage.out | tail -1 | awk '{print $NF}' | tr -d '%')
echo "statement coverage: $total%"
awk -v t="$total" -v m="${MIN_LINES:-80}" 'BEGIN { exit (t + 0 < m + 0) }' || { echo "below ${MIN_LINES:-80}%" >&2; exit 1; }

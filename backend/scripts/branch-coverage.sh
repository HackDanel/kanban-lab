#!/bin/sh
# Branch coverage of the backend, measured with gobco.
#
# gobco instruments one package at a time and only runs the tests located in
# that package's directory. The service and HTTP layers are exercised mostly by
# the integration suite in tests/integration, so for each of them the suite is
# copied next to the package as an external test package for the duration of
# the measurement.
set -eu
cd "$(dirname "$0")/.."

GOBCO="go run github.com/rillig/gobco@v1.3.4"
total_hit=0
total_all=0

measure() {
	pkg=$1
	line=$($GOBCO -branch "./$pkg" 2>/dev/null | grep '^Branch coverage:' | tail -1)
	hit=$(echo "$line" | sed -E 's/.*: ([0-9]+)\/([0-9]+).*/\1/')
	all=$(echo "$line" | sed -E 's/.*: ([0-9]+)\/([0-9]+).*/\2/')
	printf '%-28s %5s / %-5s\n' "$pkg" "$hit" "$all"
	total_hit=$((total_hit + hit))
	total_all=$((total_all + all))
}

with_integration_suite() {
	pkg=$1
	name=$(go list -f "{{.Name}}" "./$pkg")
	copied=""
	for f in tests/integration/*_test.go; do
		dst="$pkg/zz_integration_$(basename "$f")"
		sed "s/^package integration$/package ${name}_test/" "$f" > "$dst"
		copied="$copied $dst"
	done
	trap 'rm -f $copied' EXIT
	measure "$pkg"
	rm -f $copied
	trap - EXIT
}

for pkg in internal/auth internal/config internal/domain internal/platform/storage internal/realtime; do
	measure "$pkg"
done
with_integration_suite internal/service
with_integration_suite internal/http

percent=$((100 * total_hit / total_all))
echo "total: $total_hit / $total_all branches ($percent%)"
MIN_BRANCH=${MIN_BRANCH:-70}
if [ "$percent" -lt "$MIN_BRANCH" ]; then
	echo "branch coverage $percent% is below $MIN_BRANCH%" >&2
	exit 1
fi

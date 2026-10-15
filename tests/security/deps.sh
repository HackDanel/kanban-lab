#!/bin/sh
# Dependency and static analysis: govulncheck and gosec for Go, npm audit for
# the frontend and the E2E tooling. Fails on known vulnerabilities.
set -eu
cd "$(dirname "$0")/../.."
GOPROXY=${GOPROXY:-https://proxy.golang.org|https://goproxy.io|direct}
export GOPROXY

echo "== govulncheck"
(cd backend && go run golang.org/x/vuln/cmd/govulncheck@latest ./...)

echo "== gosec"
(cd backend && go run github.com/securego/gosec/v2/cmd/gosec@latest -quiet -exclude-generated -exclude-dir=tests ./...)

echo "== npm audit (frontend)"
(cd frontend && npm audit --audit-level=high --omit=dev)

echo "== npm audit (e2e)"
(cd e2e && npm audit --audit-level=high)

#!/bin/sh
# OWASP ZAP: passive baseline scan of the web UI and an active API scan driven
# by the OpenAPI contract. Reports go to tests/security/reports/.
# Usage: tests/security/zap-baseline.sh [base-url]   (default http://localhost:8000)
set -eu
cd "$(dirname "$0")"
TARGET=${1:-http://localhost:8000}
IMAGE=${ZAP_IMAGE:-ghcr.io/zaproxy/zaproxy:stable}
mkdir -p reports
chmod 777 reports

# A logged-in token lets the API scan reach the protected endpoints.
TOKEN=$(curl -sf -H 'Content-Type: application/json' \
  -d "{\"email\":\"${ADMIN_EMAIL:-admin@example.com}\",\"password\":\"${ADMIN_PASSWORD:-admin-password}\"}" \
  "$TARGET/api/v1/auth/login" | sed -E 's/.*"accessToken":"([^"]+)".*/\1/')

status=0
docker run --rm --network host -v "$PWD:/zap/wrk:rw" "$IMAGE" \
  zap-baseline.py -t "$TARGET" -c zap-rules.tsv -I \
  -r reports/zap-baseline.html -J reports/zap-baseline.json || status=$?

docker run --rm --network host -v "$PWD:/zap/wrk:rw" "$IMAGE" \
  zap-api-scan.py -t "$TARGET/docs/openapi.json" -f openapi -O "$TARGET/api/v1" -c zap-rules.tsv -I \
  -z "-config replacer.full_list(0).description=auth -config replacer.full_list(0).enabled=true -config replacer.full_list(0).matchtype=REQ_HEADER -config replacer.full_list(0).matchstr=Authorization -config replacer.full_list(0).replacement=Bearer\ $TOKEN" \
  -r reports/zap-api.html -J reports/zap-api.json || status=$?

# -I keeps warnings from failing the run; the gate is "no High alerts".
high=$(cat reports/zap-*.json | grep -o '"riskcode": *"3"' | wc -l)
echo "ZAP High alerts: $high"
[ "$high" -eq 0 ] || exit 1
exit 0

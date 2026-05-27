#!/usr/bin/env bash
# Post-deploy smoke test. Runs a curated set of probes against a deployed
# rss-fresh and exits 0 if all pass.
#
#   BASE=http://127.0.0.1:8088 OPENCLAW_GATEWAY_TOKEN=<token> ./scripts/verify-deploy.sh
#
# Add `-x` to debug.
set -euo pipefail

BASE="${BASE:-http://127.0.0.1:8088}"
TOKEN="${OPENCLAW_GATEWAY_TOKEN:-}"
fail=0

ok()   { printf '  \033[32m✔\033[0m %s\n' "$1"; }
bad()  { printf '  \033[31m✗\033[0m %s\n' "$1"; fail=1; }
note() { printf '\033[36m▸\033[0m %s\n' "$1"; }

note "1. health"
status=$(curl -fsS -o /dev/null -w '%{http_code}' "$BASE/api/v1/healthz" || echo 000)
[[ "$status" == "200" ]] && ok "healthz 200" || bad "healthz $status"

note "2. categories endpoint reachable"
curl -fsS "$BASE/api/v1/categories" >/dev/null && ok "GET /categories" || bad "GET /categories failed"

note "3. articles endpoint reachable"
curl -fsS "$BASE/api/v1/articles?limit=1" >/dev/null && ok "GET /articles" || bad "GET /articles failed"

note "4. OpenClaw endpoint requires bearer"
status=$(curl -fsS -o /dev/null -w '%{http_code}' "$BASE/api/v1/news/summary" || echo 000)
[[ "$status" == "401" ]] && ok "no token -> 401" || bad "no token returned $status (expected 401)"

note "5. OpenClaw endpoint accepts the bearer"
if [[ -z "$TOKEN" ]]; then
  bad "OPENCLAW_GATEWAY_TOKEN env var is empty — skipping authed probe"
else
  status=$(curl -fsS -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/news/summary?limit=1" || echo 000)
  [[ "$status" == "200" ]] && ok "valid token -> 200" || bad "valid token returned $status (expected 200)"
fi

note "6. CORS preflight"
status=$(curl -fsS -o /dev/null -w '%{http_code}' -X OPTIONS "$BASE/api/v1/categories" \
  -H 'Origin: https://example.test' -H 'Access-Control-Request-Method: GET' || echo 000)
[[ "$status" == "204" ]] && ok "OPTIONS -> 204" || bad "OPTIONS -> $status (expected 204)"

if [[ "$fail" -eq 0 ]]; then
  echo
  echo "All checks passed."
  exit 0
fi

echo
echo "FAILED — see above."
exit 1

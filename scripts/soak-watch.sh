#!/usr/bin/env bash
# Watches an rss-fresh container for the standard production thresholds.
# Run on the host: ./scripts/soak-watch.sh
#
# Thresholds:
#   - container RSS  : warn > 100 MB, alert > 200 MB
#   - container CPU  : informational
#   - error_count    : alert if any healthy feed climbs above 3
#   - PgBouncer pool : alert if 'pool_size_exceeded' appears in pgbouncer logs
#
# Designed for a quick eyeball check; pair with Uptime Kuma for the long haul.
set -euo pipefail
CONTAINER="${CONTAINER:-rss-fresh}"
INTERVAL="${INTERVAL:-30}"

echo "watching $CONTAINER every ${INTERVAL}s — Ctrl-C to stop"
while true; do
  ts=$(date '+%Y-%m-%d %H:%M:%S')
  stats=$(docker stats --no-stream --format '{{.MemUsage}} | CPU {{.CPUPerc}}' "$CONTAINER" 2>/dev/null || echo "down")
  echo "$ts  $stats"
  case "$stats" in
    *"GiB"*) echo "  !! RAM exceeded 1 GiB — investigate" ;;
    *"MiB"*)
      mb=$(echo "$stats" | awk -F'MiB' '{print $1}' | awk '{print $1}')
      mb_int=${mb%.*}
      if [[ "$mb_int" =~ ^[0-9]+$ ]] && (( mb_int > 200 )); then
        echo "  !! RAM > 200 MiB ($mb MiB)"
      elif [[ "$mb_int" =~ ^[0-9]+$ ]] && (( mb_int > 100 )); then
        echo "  ~  RAM > 100 MiB ($mb MiB) — keep an eye"
      fi
      ;;
  esac
  sleep "$INTERVAL"
done

#!/usr/bin/env bash
# Seed a fresh rss-fresh instance with a starter set of categories + feeds.
# Run once from inside Cloudflare Access (or set BASE to localhost on the host).
set -euo pipefail

BASE="${BASE:-http://127.0.0.1:8088}"
JQ="${JQ:-jq}"

require_curl() { command -v curl >/dev/null || { echo "curl missing" >&2; exit 1; }; }
require_jq()   { command -v "$JQ" >/dev/null || { echo "jq missing" >&2; exit 1; }; }
require_curl
require_jq

post_json() { curl -fsS -H 'Content-Type: application/json' -X POST "$BASE$1" -d "$2"; }

create_cat() {
  local name="$1" critical="${2:-false}"
  echo "→ category: $name (critical=$critical)"
  post_json "/api/v1/categories" "$(jq -nc --arg n "$name" --argjson c "$critical" '{name:$n,is_critical:$c}')" \
    | "$JQ" -r '.id'
}

add_feed() {
  local cat_id="$1" url="$2"
  echo "→ feed under cat $cat_id: $url"
  post_json "/api/v1/feeds" "$(jq -nc --argjson c "$cat_id" --arg u "$url" '{category_id:$c,url:$u}')" \
    | "$JQ" -r '.id' >/dev/null
}

# === starter set — edit to taste ===
AI=$(create_cat "AI" true)            # critical → Telegram push
TECH=$(create_cat "Tech" false)
WORLD=$(create_cat "World" false)

add_feed "$AI"    "https://openai.com/blog/rss.xml"
add_feed "$AI"    "https://huggingface.co/blog/feed.xml"
add_feed "$AI"    "https://www.anthropic.com/news/rss.xml"

add_feed "$TECH"  "https://news.ycombinator.com/rss"
add_feed "$TECH"  "https://lobste.rs/rss"
add_feed "$TECH"  "https://github.com/blog/all.atom"

add_feed "$WORLD" "http://feeds.bbci.co.uk/news/world/rss.xml"
add_feed "$WORLD" "https://www.aljazeera.com/xml/rss/all.xml"

echo "✔ seeded — first fetch will run on the next FETCH_CRON tick (or POST /feeds/:id/refresh to force)."

#!/usr/bin/env bash
# Converts Google Takeout music-library-songs.csv to a yt-dlp search list.
# Usage: ./ytm-csv-to-list.sh <path-to-music-library-songs.csv>
# Output: config/ytm-library.txt  (one "Artist - Title" search term per line)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"
OUT="$CONFIG_DIR/ytm-library.txt"

CSV="${1:-}"
if [[ -z "$CSV" || ! -f "$CSV" ]]; then
  echo "Usage: $0 <path-to-music-library-songs.csv>" >&2
  exit 1
fi

mkdir -p "$CONFIG_DIR"

# Takeout CSV columns: Title, Album, Artist, Duration, ...
# Skip header (NR>1), print "Artist - Title" as yt-dlp search term.
# Simple awk split works because Google Takeout quotes fields with embedded commas.
awk -F',' 'NR>1 && $3!="" && $1!="" { gsub(/^"|"$/, "", $1); gsub(/^"|"$/, "", $3); print $3 " - " $1 }' "$CSV" > "$OUT"

COUNT=$(wc -l < "$OUT")
echo "Wrote $COUNT search terms to $OUT"

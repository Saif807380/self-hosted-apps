#!/usr/bin/env bash
# Extracts video IDs from Google Takeout YouTube Music playlist CSVs and
# writes them as YouTube URLs to config/ytm-library.txt (deduplicated).
#
# Usage:
#   ./ytm-playlists-to-list.sh <takeout-playlists-dir>
#   ./ytm-playlists-to-list.sh <single-playlist-videos.csv>
#
# Each Takeout playlist CSV looks like:
#   Video ID,Playlist video creation timestamp,Playlist video update timestamp
#   dQw4w9WgXcQ,2024-01-01T00:00:00+00:00,2024-01-01T00:00:00+00:00
#   ...
# We grab column 1, validate it's an 11-char video ID, and emit a URL.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"
OUT="$CONFIG_DIR/ytm-library.txt"

INPUT="${1:-}"
if [[ -z "$INPUT" ]]; then
  echo "Usage: $0 <takeout-playlists-dir-or-single-csv>" >&2
  exit 1
fi

mkdir -p "$CONFIG_DIR"

declare -a CSV_FILES
if [[ -d "$INPUT" ]]; then
  while IFS= read -r -d '' f; do CSV_FILES+=("$f"); done \
    < <(find "$INPUT" -maxdepth 2 -type f -name '*.csv' -print0)
elif [[ -f "$INPUT" ]]; then
  CSV_FILES=("$INPUT")
else
  echo "Not a file or directory: $INPUT" >&2
  exit 1
fi

if [[ ${#CSV_FILES[@]} -eq 0 ]]; then
  echo "No CSV files found in $INPUT" >&2
  exit 1
fi

# Match lines starting with an 11-char YouTube video ID followed by comma.
# Skips header rows, blank lines, and any metadata blocks Takeout sometimes prepends.
{
  for csv in "${CSV_FILES[@]}"; do
    awk -F',' '/^[A-Za-z0-9_-]{11},/ { print $1 }' "$csv"
  done
} | sort -u | sed 's|^|https://music.youtube.com/watch?v=|' > "$OUT"

COUNT=$(wc -l < "$OUT")
echo "Wrote $COUNT unique video URLs to $OUT (from ${#CSV_FILES[@]} playlist CSV(s))"

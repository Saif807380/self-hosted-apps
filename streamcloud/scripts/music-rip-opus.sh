#!/usr/bin/env bash
# Bulk Opus rip from YouTube for all tracks in config/ytm-library.txt.
# Safe to re-run: already-downloaded tracks are skipped via the archive file.
# Usage: ./music-rip-opus.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"
BATCH="$CONFIG_DIR/ytm-library.txt"
ARCHIVE="$CONFIG_DIR/music-archive.txt"
DEST="/home/saifkazi/Music/"

if [[ ! -f "$BATCH" ]]; then
  echo "No batch file found at $BATCH" >&2
  echo "Run ytm-csv-to-list.sh first to generate it from your Google Takeout CSV." >&2
  exit 1
fi

echo "Starting Opus rip — $(wc -l < "$BATCH") search terms"
echo "Downloads go to: $DEST"
echo "Archive (dedup): $ARCHIVE"
echo ""

yt-dlp \
  --cookies-from-browser firefox \
  --batch-file="$BATCH" \
  --default-search "ytsearch1" \
  -x --audio-format opus --audio-quality 0 \
  --embed-metadata --embed-thumbnail \
  --parse-metadata "artist:(?P<album_artist>[^,]+)" \
  --download-archive "$ARCHIVE" \
  -o "$DEST/%(title)s.%(ext)s" \
  --no-playlist \
  --sleep-interval 1 \
  --ignore-errors
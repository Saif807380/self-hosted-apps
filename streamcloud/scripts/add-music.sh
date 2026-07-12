#!/usr/bin/env bash
# Single-command add-to-library wrapper.
# Usage: add-music.sh <url> [<url>...]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"
ARCHIVE="$CONFIG_DIR/music-archive.txt"
DEST="$HOME/Music"

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <url> [<url>...]" >&2
  exit 1
fi

mkdir -p "$DEST"

DL_ARGS=()
if [[ "${AUTO_IMPORT:-0}" == "1" ]]; then
  DL_ARGS=(--no-download-archive)
fi

echo "Downloading $# URL(s) to $DEST"

# yt-dlp exits non-zero when any single item fails (unavailable video, thumbnail
# postprocessing, etc.) even with --ignore-errors. Under `set -e` that would abort
# the script before the tag/move pipeline runs, stranding good downloads in $DEST.
# `|| echo ...` swallows the non-zero exit so we always proceed to fix_tags.py.
yt-dlp \
  --cookies-from-browser firefox \
  -x --audio-format opus --audio-quality 0 \
  --embed-metadata --embed-thumbnail \
  --parse-metadata "artist:(?P<album_artist>[^,]+)" \
  --download-archive "$ARCHIVE" \
  "${DL_ARGS[@]}" \
  -o "$DEST/%(title)s.%(ext)s" \
  --sleep-interval 1 \
  --ignore-errors \
  "$@" || echo "yt-dlp reported errors (continuing to tag/move what downloaded)"

# Remove postprocessing leftovers so they never enter the tagging pipeline.
rm -f "$DEST"/*.temp.* "$DEST"/*.part 2>/dev/null || true

echo ""
echo "Running full Addition Pipeline (Metadata + Tagging + Move)..."
python3 "$SCRIPT_DIR/fix_tags.py"

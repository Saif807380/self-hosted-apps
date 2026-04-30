#!/usr/bin/env bash
# Single-command add-to-library wrapper.
# Usage: add-music.sh <url> [<url>...]
#
# Each URL can be a single track, an album playlist, or a YT Music playlist.
# yt-dlp downloads to ~/Music/ as Opus 192k, then `beet import` is run
# interactively so you can confirm matches and tags before files move into
# /srv/media/music/.
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

echo "Downloading $# URL(s) to $DEST"

yt-dlp \
  --cookies-from-browser firefox \
  -x --audio-format opus --audio-quality 0 \
  --embed-metadata --embed-thumbnail \
  --parse-metadata "artist:(?P<album_artist>[^,]+)" \
  --download-archive "$ARCHIVE" \
  -o "$DEST/%(title)s.%(ext)s" \
  --sleep-interval 1 \
  --ignore-errors \
  "$@"

echo ""
read -r -p "Run 'beet import $DEST' now? [Y/n] " ans
if [[ -z "$ans" || "$ans" =~ ^[Yy]$ ]]; then
  beet import -A -s "$DEST"
else
  echo "Skipped. Run \`beet import $DEST\` later when ready."
fi

#!/usr/bin/env bash
# Generate ListenBrainz auto-playlists via troi, convert to M3U for Navidrome.
#
# Personalised patches (daily-jams, weekly-jams, weekly-exploration) need ~2-3
# weeks of scrobble history before they produce useful results — they'll be
# empty or sparse until then. Re-runs are cheap; cron weekly via the .timer.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Load env (LISTENBRAINZ_USER, NAVIDROME_USER, NAVIDROME_PASS, NAVIDROME_URL).
if [[ -f "$PROJECT_DIR/.env" ]]; then
  set -a; source "$PROJECT_DIR/.env"; set +a
fi

: "${LISTENBRAINZ_USER:?Set LISTENBRAINZ_USER in streamcloud/.env}"
: "${NAVIDROME_USER:?Set NAVIDROME_USER in streamcloud/.env}"
: "${NAVIDROME_PASS:?Set NAVIDROME_PASS in streamcloud/.env}"
export NAVIDROME_URL="${NAVIDROME_URL:-http://127.0.0.1:4533}"
export MUSIC_FOLDER="${MUSIC_FOLDER:-/srv/media/music}"
export NAVIDROME_USER NAVIDROME_PASS

PLAYLIST_DIR="$MUSIC_FOLDER/Playlists"
TMP_DIR="$(mktemp -d)"
trap "rm -rf $TMP_DIR" EXIT

mkdir -p "$PLAYLIST_DIR"

if [[ $# -eq 0 ]]; then
  echo "Usage: $0 <patch_name> [<patch_name>...]" >&2
  exit 1
fi

PATCHES=("$@")

for patch in "${PATCHES[@]}"; do
  jspf="$TMP_DIR/$patch.jspf"
  m3u="$PLAYLIST_DIR/$patch.m3u"

  echo "→ Running patch: $patch"
  cd "$TMP_DIR" || exit 1
  rm -f playlist_000.jspf

  if troi playlist -u -c "$LISTENBRAINZ_USER" -s periodic-jams "$LISTENBRAINZ_USER" "$patch" || \
     troi playlist -s periodic-jams "$LISTENBRAINZ_USER" "$patch"; then
     
    if [[ -f "playlist_000.jspf" ]]; then
      mv "playlist_000.jspf" "$jspf"
    fi

    if [[ -s "$jspf" ]]; then
      AUTO_DOWNLOAD=1 "$SCRIPT_DIR/jspf-to-m3u.py" "$jspf" > "$m3u" \
        && echo "  ✓ wrote $m3u" \
        || echo "  ! converter failed"
    else
      echo "  - empty playlist (insufficient scrobble history?)"
    fi
  else
    echo "  ! troi failed for $patch (CLI mismatch or insufficient data); skipping"
  fi
done

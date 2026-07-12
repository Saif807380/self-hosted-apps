#!/usr/bin/env bash
# Shared helpers for video auto-prune scripts. Sourced by:
#   prune-video.sh  (Layer A: daily watched-and-delete)
#   disk-prune.sh   (Layer B: hourly hard-ceiling guard)

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -f "$PROJECT_ROOT/.env" ]]; then
  set -a; source "$PROJECT_ROOT/.env"; set +a
fi

: "${JELLYFIN_URL:=http://127.0.0.1:8096}"
: "${SONARR_URL:=http://127.0.0.1:8989}"
: "${RADARR_URL:=http://127.0.0.1:7878}"
: "${JELLYFIN_USER:=}"
: "${VIDEO_ROOT:=/srv/media/video}"
: "${DRY_RUN:=1}"

# Completed downloads land here and are hardlinked into the library by *arr.
DOWNLOADS_DIR="$VIDEO_ROOT/downloads"

_log() { printf '[%s] %s\n' "$(date -Is)" "$*"; }

_require() {
  for cmd in "$@"; do
    command -v "$cmd" >/dev/null || { _log "FATAL: missing command '$cmd'"; exit 1; }
  done
}
_require curl jq du awk date flock find stat

: "${JELLYFIN_API_KEY:?JELLYFIN_API_KEY missing — fill in $PROJECT_ROOT/.env}"
: "${SONARR_API_KEY:?SONARR_API_KEY missing — fill in $PROJECT_ROOT/.env}"
: "${RADARR_API_KEY:?RADARR_API_KEY missing — fill in $PROJECT_ROOT/.env}"

# Single-instance lock shared between both prune scripts.
acquire_lock() {
  local lockfile="${XDG_RUNTIME_DIR:-/tmp}/streamcloud-prune.lock"
  exec 200>"$lockfile"
  flock -n 200 || { _log "another prune is already running, exiting"; exit 0; }
}

jf_user_id() {
  local users
  users=$(curl -fsS "$JELLYFIN_URL/Users" -H "X-Emby-Token: $JELLYFIN_API_KEY")
  if [[ -n "$JELLYFIN_USER" ]]; then
    jq -er --arg n "$JELLYFIN_USER" '.[] | select(.Name == $n) | .Id' <<<"$users" \
      || { _log "FATAL: Jellyfin user '$JELLYFIN_USER' not found"; exit 1; }
  else
    jq -er '.[0].Id' <<<"$users"
  fi
}

# List watched Movie + Episode items for the given user, oldest-played first.
# Output: one JSON object per line with .Path .Type .Name .SeriesName .LastPlayed
jf_watched_items() {
  local user_id="$1"
  curl -fsS "$JELLYFIN_URL/Users/$user_id/Items" --get \
    --data-urlencode "Recursive=true" \
    --data-urlencode "IsPlayed=true" \
    --data-urlencode "IncludeItemTypes=Movie,Episode" \
    --data-urlencode "Fields=Path,UserData" \
    --data-urlencode "SortBy=DatePlayed" \
    --data-urlencode "SortOrder=Ascending" \
    -H "X-Emby-Token: $JELLYFIN_API_KEY" \
    | jq -c '.Items[] | {
        Path: .Path,
        Type: .Type,
        Name: .Name,
        SeriesName: .SeriesName,
        LastPlayed: .UserData.LastPlayedDate
      } | select(.Path != null and .LastPlayed != null)'
}

# path -> movieFile.id map for Radarr.
radarr_path_map() {
  curl -fsS "$RADARR_URL/api/v3/movie" -H "X-Api-Key: $RADARR_API_KEY" \
    | jq 'map(select(.movieFile.path != null))
          | map({key: .movieFile.path, value: .movieFile.id})
          | from_entries'
}

# path -> episodeFile.id map for Sonarr (across all series).
sonarr_path_map() {
  local series_ids merged='{}' per
  series_ids=$(curl -fsS "$SONARR_URL/api/v3/series" -H "X-Api-Key: $SONARR_API_KEY" | jq -r '.[].id')
  for sid in $series_ids; do
    per=$(curl -fsS "$SONARR_URL/api/v3/episodefile?seriesId=$sid" -H "X-Api-Key: $SONARR_API_KEY" \
            | jq 'map({key: .path, value: .id}) | from_entries')
    merged=$(jq -n --argjson a "$merged" --argjson b "$per" '$a * $b')
  done
  echo "$merged"
}

radarr_delete_file() {
  local id="$1"
  if [[ "$DRY_RUN" == "1" ]]; then
    _log "  [dry-run] would DELETE Radarr movieFile $id"
    return 0
  fi
  curl -fsS -X DELETE "$RADARR_URL/api/v3/moviefile/$id" -H "X-Api-Key: $RADARR_API_KEY" >/dev/null \
    || { _log "  Radarr DELETE failed for movieFile $id"; return 1; }
}

sonarr_delete_file() {
  local id="$1"
  if [[ "$DRY_RUN" == "1" ]]; then
    _log "  [dry-run] would DELETE Sonarr episodeFile $id"
    return 0
  fi
  curl -fsS -X DELETE "$SONARR_URL/api/v3/episodefile/$id" -H "X-Api-Key: $SONARR_API_KEY" >/dev/null \
    || { _log "  Sonarr DELETE failed for episodeFile $id"; return 1; }
}

# A library file deleted by *arr is hardlinked to a copy in $DOWNLOADS_DIR, so
# its data isn't freed until that copy is removed too. This removes it, given the
# inode of the (now-deleted) library file. Idempotent: if nothing in downloads
# holds that inode (already gone, or was a plain copy), it does nothing.
delete_download_hardlink() {
  local ino="$1" dlfile
  dlfile=$(find "$DOWNLOADS_DIR" -inum "$ino" -print -quit 2>/dev/null)
  [[ -z "$dlfile" ]] && return 0

  if [[ "$DRY_RUN" == "1" ]]; then
    _log "  [dry-run] would delete download copy: $dlfile"
    return 0
  fi

  rm -f -- "$dlfile" \
    && _log "  reclaimed download copy: $dlfile" \
    || { _log "  failed to delete download copy: $dlfile"; return 1; }
  # Drop the release folder too if this was its last file.
  rmdir --ignore-fail-on-non-empty "$(dirname -- "$dlfile")" 2>/dev/null || true
}

# Delete one watched item via the appropriate *arr API, then remove its
# hardlinked download copy. Returns 0 on success, 1 if no match (caller should
# keep going).
delete_item() {
  local item="$1" rmap="$2" smap="$3"
  local type path name id ino
  type=$(jq -r '.Type' <<<"$item")
  path=$(jq -r '.Path' <<<"$item")
  name=$(jq -r '.Name + (if .SeriesName then " — \(.SeriesName)" else "" end)' <<<"$item")
  # Capture the inode before deletion so we can find its download hardlink after.
  ino=$(stat -c %i "$path" 2>/dev/null || true)

  case "$type" in
    Movie)
      id=$(jq -r --arg p "$path" '.[$p] // empty' <<<"$rmap")
      if [[ -z "$id" ]]; then _log "  no Radarr match for $path"; return 1; fi
      radarr_delete_file "$id" || return 1
      _log "deleted movie: $name"
      ;;
    Episode)
      id=$(jq -r --arg p "$path" '.[$p] // empty' <<<"$smap")
      if [[ -z "$id" ]]; then _log "  no Sonarr match for $path"; return 1; fi
      sonarr_delete_file "$id" || return 1
      _log "deleted episode: $name"
      ;;
    *)
      _log "  unknown item type: $type"; return 1
      ;;
  esac

  [[ -n "$ino" ]] && delete_download_hardlink "$ino"
  return 0
}

# Integer GiB usage of $VIDEO_ROOT (rounded up by `du -BG`).
video_usage_gb() {
  du -BG -s "$VIDEO_ROOT" 2>/dev/null | awk '{ sub("G","",$1); print int($1) }'
}

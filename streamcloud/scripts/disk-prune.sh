#!/usr/bin/env bash
# Layer B: when /srv/media/video exceeds HIGH_WATER_GB, delete oldest-watched
# items until back under LOW_WATER_GB. Stops if it runs out of watched items
# (does NOT touch unwatched content).

source "$(dirname "${BASH_SOURCE[0]}")/prune-lib.sh"

: "${HIGH_WATER_GB:=95}"
: "${LOW_WATER_GB:=90}"

acquire_lock

usage=$(video_usage_gb)
_log "$VIDEO_ROOT usage: ${usage}G  (high=${HIGH_WATER_GB}G low=${LOW_WATER_GB}G)"
(( usage < HIGH_WATER_GB )) && exit 0

[[ "$DRY_RUN" == "1" ]] && _log "DRY_RUN=1 — no deletions. Set DRY_RUN=0 in .env to enable."

user_id=$(jf_user_id)
items=$(jf_watched_items "$user_id")
count=$(printf '%s\n' "$items" | grep -c . || true)
_log "$count watched items available, oldest-played first"

if (( count == 0 )); then
  _log "WARN: over high-water with nothing watched to prune — manual cleanup needed"
  exit 1
fi

_log "building Radarr/Sonarr path indexes…"
rmap=$(radarr_path_map)
smap=$(sonarr_path_map)

deleted=0
while IFS= read -r item; do
  delete_item "$item" "$rmap" "$smap" || continue
  deleted=$((deleted+1))
  # In dry-run, du won't drop — break to avoid spamming the entire watched list.
  if [[ "$DRY_RUN" == "1" ]]; then
    _log "  [dry-run] stopping after first match (would normally re-check disk usage)"
    break
  fi
  usage=$(video_usage_gb)
  if (( usage < LOW_WATER_GB )); then
    _log "reached low-water mark at ${usage}G after deleting $deleted item(s)"
    exit 0
  fi
done <<<"$items"

[[ "$DRY_RUN" == "1" ]] && exit 0

_log "WARN: deleted $deleted item(s), still at ${usage}G"
exit 1

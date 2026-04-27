#!/usr/bin/env bash
# Layer A: delete movies/episodes the configured Jellyfin user watched
# more than PRUNE_AGE_DAYS ago. Default age is 14 days, default DRY_RUN=1.

source "$(dirname "${BASH_SOURCE[0]}")/prune-lib.sh"

: "${PRUNE_AGE_DAYS:=14}"

acquire_lock

[[ "$DRY_RUN" == "1" ]] && _log "DRY_RUN=1 — no deletions. Set DRY_RUN=0 in .env to enable."

user_id=$(jf_user_id)
_log "Jellyfin user id: $user_id"

cutoff=$(date -u -d "$PRUNE_AGE_DAYS days ago" +%Y-%m-%dT%H:%M:%S.0000000Z)
_log "deleting items watched before $cutoff (older than ${PRUNE_AGE_DAYS}d)"

items=$(jf_watched_items "$user_id" | jq -c --arg c "$cutoff" 'select(.LastPlayed < $c)')
count=$(printf '%s\n' "$items" | grep -c . || true)
_log "found $count eligible items"
(( count == 0 )) && exit 0

_log "building Radarr/Sonarr path indexes…"
rmap=$(radarr_path_map)
smap=$(sonarr_path_map)

deleted=0; missed=0
while IFS= read -r item; do
  if delete_item "$item" "$rmap" "$smap"; then
    deleted=$((deleted+1))
  else
    missed=$((missed+1))
  fi
done <<<"$items"

_log "done: $deleted deleted, $missed unmatched"

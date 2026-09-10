#!/usr/bin/env bash
# Install the cortex Quadlets into the user systemd manager.
#
# Copies rather than symlinks: the Quadlet generator runs early in the user
# manager's startup, and a symlink into a home directory that isn't mounted yet
# is a failure mode nobody enjoys debugging. Matches how streamcloud/systemd-user
# is already deployed.
#
# Rerunnable. No sudo -- everything here is rootless and user-scoped.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
QUADLET_DIR="$HOME/.config/containers/systemd"
DATA_DIR="$HOME/.local/share/cortex/open-webui"
SEARXNG_RUNTIME="$HOME/.local/share/cortex/searxng"

ENV_FILES=(cortex/.env.searxng cortex/.env.open-webui)

for rel in "${ENV_FILES[@]}"; do
  f="$REPO/$rel"
  if [[ ! -f "$f" ]]; then
    echo "$rel is missing. See cortex/.env.example for what goes in it." >&2
    exit 1
  fi
  # The Quadlets hand these to podman as --env-file. Any mode wider than 600
  # means every user on the box can read the secrets in them.
  mode=$(stat -c '%a' "$f")
  if [[ "$mode" != "600" ]]; then
    echo "$rel is mode $mode; tightening to 600." >&2
    chmod 600 "$f"
  fi
done

mkdir -p "$QUADLET_DIR"

# Open WebUI's state is a SQLite DB plus a Chroma vector store: many small
# random writes, which is precisely what btrfs copy-on-write fragments. The
# flag only affects files created after it is set, so it has to happen while
# the directory is still empty -- and it is inherited by everything created
# inside afterwards.
if [[ ! -d "$DATA_DIR" ]]; then
  mkdir -p "$DATA_DIR"
  if chattr +C "$DATA_DIR" 2>/dev/null; then
    echo "Set +C (no CoW) on $DATA_DIR"
  else
    echo "Could not set +C on $DATA_DIR -- not btrfs, or unsupported. Continuing." >&2
  fi
fi

# Render SearXNG's settings.yml with the secret substituted in.
#
# The tracked template carries a @SEARXNG_SECRET@ placeholder so no secret ever
# reaches git. Rendering it here rather than letting the image's entrypoint
# rewrite the file in place is deliberate: the entrypoint's sed needs a writable
# settings.yml, which would mean either mounting the repo file read-write and
# watching a secret get written into a tracked path, or dropping :ro on the
# mount. This keeps the mount read-only and the repo clean.
mkdir -p "$SEARXNG_RUNTIME"
secret=$(grep -E '^SEARXNG_SECRET=' "$REPO/cortex/.env.searxng" | cut -d= -f2-)
if [[ -z "$secret" ]]; then
  echo "SEARXNG_SECRET is empty in cortex/.env.searxng. Generate one: openssl rand -hex 32" >&2
  exit 1
fi
umask 077
sed "s|@SEARXNG_SECRET@|$secret|" \
  "$REPO/cortex/searxng/settings.yml" > "$SEARXNG_RUNTIME/settings.yml"
# The container runs SearXNG as its own unprivileged user, which maps into the
# subuid range -- not to this user. It has to be able to read the file.
chmod 644 "$SEARXNG_RUNTIME/settings.yml"
umask 022
echo "Rendered $SEARXNG_RUNTIME/settings.yml"

for unit in "$REPO"/cortex/quadlets/*.container; do
  install -m 644 "$unit" "$QUADLET_DIR/$(basename "$unit")"
  echo "Installed $(basename "$unit")"
done

systemctl --user daemon-reload

echo
echo "Generated units:"
systemctl --user list-unit-files 'open-webui.service' 'searxng.service' --no-pager 2>/dev/null || true
/usr/lib/systemd/user-generators/podman-user-generator --dryrun 2>&1 | grep -E '^---' || true

cat <<'NEXT'

Next:
  systemctl --user start searxng open-webui
  systemctl --user status searxng open-webui
NEXT

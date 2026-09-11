#!/usr/bin/env bash
# Install (or reinstall) the Beszel hub Quadlet and the agent user unit.
#
# Idempotent and safe to re-run: it regenerates nothing that already exists,
# and it never touches the hub's database. Needs no sudo at any point.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
QUADLET_DIR="$HOME/.config/containers/systemd"
UNIT_DIR="$HOME/.config/systemd/user"
DATA_DIR="$HOME/.local/share/beszel"
HUB_ENV="$REPO/.env.beszel-hub"
AGENT_ENV="$REPO/.env.beszel-agent"
AGENT_BIN="$HOME/.local/bin/beszel-agent"
AGENT_VERSION="0.19.0"

say() { printf '\n== %s\n' "$*"; }

# --------------------------------------------------------------- agent binary
# Deliberately ~/.local/bin and not the AUR: the AUR package installs a SYSTEM
# service running as a `beszel` user, which cannot read the rootless podman
# socket at /run/user/1000/podman/podman.sock. Updating is `beszel-agent
# update` -- a self-update that also needs no root.
if [[ ! -x "$AGENT_BIN" ]]; then
  say "downloading beszel-agent $AGENT_VERSION"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  curl -fsSL -o "$tmp/agent.tar.gz" \
    "https://github.com/henrygd/beszel/releases/download/v${AGENT_VERSION}/beszel-agent_linux_amd64.tar.gz"
  tar xzf "$tmp/agent.tar.gz" -C "$tmp"
  mkdir -p "$(dirname "$AGENT_BIN")"
  install -m 755 "$tmp/beszel-agent" "$AGENT_BIN"
fi
echo "agent: $("$AGENT_BIN" --version)"

# ------------------------------------------------------------------ podman API
# Beszel reads container stats over the Docker Engine API, which podman serves
# on this socket. It is NOT enabled by default on Arch. Without it the
# Containers tab is empty and the agent logs nothing you would notice.
if ! systemctl --user is-enabled podman.socket >/dev/null 2>&1; then
  say "enabling podman.socket"
  systemctl --user enable --now podman.socket
fi
test -S "/run/user/$(id -u)/podman/podman.sock" \
  || { echo "ERROR: podman socket missing after enabling podman.socket" >&2; exit 1; }

# --------------------------------------------------------------- data dirs
# chattr +C (nodatacow) must be set on an EMPTY directory -- btrfs cannot
# apply it to a directory that already holds files, and children inherit it.
# SQLite's random rewrites on a copy-on-write filesystem fragment badly
# otherwise. Same treatment cortex gives Open WebUI's database.
for d in "$DATA_DIR/hub" "$DATA_DIR/agent"; do
  if [[ ! -d "$d" ]]; then
    mkdir -p "$d"
    chattr +C "$d" 2>/dev/null || echo "  note: chattr +C failed on $d (not btrfs?)"
  fi
done

# ------------------------------------------------------------------- hub env
if [[ ! -f "$HUB_ENV" ]]; then
  say "generating $HUB_ENV"
  email="${BESZEL_USER_EMAIL:-$(git config user.email 2>/dev/null || echo "beszel@localhost")}"
  # Only ever typed at the PocketBase superuser screen (/_/). AUTO_LOGIN covers
  # the dashboard itself, so this is a break-glass value, not a daily password.
  # Not `tr </dev/urandom | head -c 32`: head closes the pipe at 32 bytes, tr
  # dies of SIGPIPE, and `set -o pipefail` turns that into an aborted install.
  password="$(python3 -c 'import secrets,string; print("".join(secrets.choice(string.ascii_letters+string.digits) for _ in range(32)))')"
  umask 077
  cat > "$HUB_ENV" <<EOF
USER_EMAIL=$email
USER_PASSWORD=$password
AUTO_LOGIN=$email
APP_URL=https://$(hostname).tail4f0f0b.ts.net:8444
EOF
fi
chmod 600 "$HUB_ENV"
# shellcheck disable=SC1090
USER_EMAIL="$(grep -E '^USER_EMAIL=' "$HUB_ENV" | cut -d= -f2-)"

# ---------------------------------------------------------------- config.yml
# Rendered into the hub's data dir, where the hub reads it on every start.
say "rendering config.yml"
sed "s|@USER_EMAIL@|$USER_EMAIL|g" "$REPO/hub/config.yml" > "$DATA_DIR/hub/config.yml"
chmod 644 "$DATA_DIR/hub/config.yml"

# ----------------------------------------------------------------- unit files
say "installing units"
mkdir -p "$QUADLET_DIR" "$UNIT_DIR"
install -m 644 "$REPO/quadlets/beszel-hub.container" "$QUADLET_DIR/"
install -m 644 "$REPO/systemd-user/beszel-agent.service" "$UNIT_DIR/"

# The agent's KEY is the hub's public key, which does not exist until the hub
# has booted once. Write a placeholder so the unit can be installed now, and
# fill it in below once the hub has generated the pair.
if [[ ! -f "$AGENT_ENV" ]]; then
  umask 077
  echo "KEY=" > "$AGENT_ENV"
fi
chmod 600 "$AGENT_ENV"

systemctl --user daemon-reload
systemctl --user enable beszel-agent.service >/dev/null
# NOTE: `systemctl --user enable beszel-hub` would fail -- Quadlet units are
# generated into /run and systemd refuses to enable a generated unit. The
# [Install] section in the .container file is what creates the autostart
# symlink, via the generator.

# ---------------------------------------------------------------- hub, then key
say "starting hub"
systemctl --user start beszel-hub.service
for _ in $(seq 1 60); do
  [[ -f "$DATA_DIR/hub/id_ed25519" ]] && break
  sleep 1
done
test -f "$DATA_DIR/hub/id_ed25519" \
  || { echo "ERROR: hub did not generate a key pair -- journalctl --user -u beszel-hub" >&2; exit 1; }

pubkey="$(ssh-keygen -y -f "$DATA_DIR/hub/id_ed25519")"
if ! grep -qxF "KEY=$pubkey" "$AGENT_ENV"; then
  say "writing hub public key into $AGENT_ENV"
  umask 077
  echo "KEY=$pubkey" > "$AGENT_ENV"
  chmod 600 "$AGENT_ENV"
fi

say "starting agent"
systemctl --user restart beszel-agent.service

say "done"
printf '  hub    http://127.0.0.1:8090\n'
printf '  agent  127.0.0.1:45876\n'
printf '\nExpose it to the tailnet with: %s\n' "$REPO/scripts/tailscale-serve.sh"

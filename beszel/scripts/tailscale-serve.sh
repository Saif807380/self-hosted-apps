#!/usr/bin/env bash
# Publish the Beszel hub on the tailnet at https://<host>.<tailnet>.ts.net:8444
#
# Port choice: :443 is navidrome and :8443 is Open WebUI, both already claimed
# on this tailnet. Check with `tailscale serve status` before changing it.
#
# This needs no sudo -- the tailscale operator is already this user.
#
# Tailscale terminates TLS with a real Let's Encrypt certificate and proxies to
# the hub's loopback port. The hub is never itself reachable off the box, which
# matters more here than usual: AUTO_LOGIN means reaching the port IS being
# logged in. Tailnet membership is the only thing standing in front of it.
set -euo pipefail

PORT=8444
TARGET=8090

if ! curl -sf --max-time 5 "http://127.0.0.1:$TARGET/api/health" >/dev/null; then
  echo "ERROR: hub not answering on 127.0.0.1:$TARGET -- start it first:" >&2
  echo "         systemctl --user start beszel-hub" >&2
  exit 1
fi

tailscale serve --bg --https="$PORT" "$TARGET"
echo
tailscale serve status

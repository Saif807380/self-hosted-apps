#!/usr/bin/env bash
# Publish Open WebUI on the tailnet over HTTPS.
#
# This does NOT start a second Tailscale node. `tailscale serve` is config on
# the daemon already running here; this adds an 8443 listener alongside the
# existing 443 -> Navidrome mapping.
#
# Why HTTPS rather than plain http://cachyos:8081, which would also work over
# the tailnet: browsers gate camera, microphone and PWA install behind a secure
# context. Pointing the phone's camera at a document and sending it to the
# vision model is a primary use case here, so the Let's Encrypt certificate
# Tailscale provisions is worth the extra port.
#
# Needs root, or `tailscale set --operator=$USER` once (see cortex/README.md).
set -euo pipefail

PORT_HTTPS=8443
PORT_LOCAL=8081

if ! curl -sf -o /dev/null "http://127.0.0.1:${PORT_LOCAL}/health"; then
  echo "Nothing is answering on 127.0.0.1:${PORT_LOCAL}." >&2
  echo "Start it first: systemctl --user start open-webui" >&2
  exit 1
fi

tailscale serve --bg --https="${PORT_HTTPS}" "${PORT_LOCAL}"

echo
tailscale serve status
echo
echo "The serve config persists across reboots -- it lives in the daemon's"
echo "state, not in a unit file, so there is nothing to enable."

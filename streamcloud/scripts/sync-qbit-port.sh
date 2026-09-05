#!/bin/sh
# Push gluetun's forwarded VPN port into qBittorrent's listening port.
#
# Runs *inside the gluetun container* (Alpine, so POSIX sh — not bash), invoked
# by VPN_PORT_FORWARDING_UP_COMMAND every time ProtonVPN assigns a port.
#
# Why: ProtonVPN hands out a different forwarded port on every reconnect, and
# gluetun only opens *that* port in its firewall. qBittorrent stores a static
# Session\Port, so after any reconnect the two drift apart, no inbound peer can
# reach qBittorrent, and it sits there "firewalled" with a starved swarm.
#
# qBittorrent has WebUI\LocalHostAuth=false, so the loopback API call needs no
# credentials. gluetun shares qBittorrent's netns, so 127.0.0.1 is qBittorrent.

set -eu

PORT_FILE=/tmp/gluetun/forwarded_port
QBIT_API=http://127.0.0.1:8080/api/v2/app/setPreferences

port="${1:-}"

# gluetun substitutes {{PORTS}}; fall back to its status file if it didn't.
case "$port" in
    '' | *[!0-9]*) port=$(cat "$PORT_FILE" 2>/dev/null || echo '') ;;
esac

case "$port" in
    '' | *[!0-9]*)
        echo "sync-qbit-port: no usable port (arg='${1:-}')" >&2
        exit 1
        ;;
esac

wget -qO- --post-data="json={\"listen_port\":$port}" "$QBIT_API" >/dev/null
echo "sync-qbit-port: qBittorrent listen port set to $port"

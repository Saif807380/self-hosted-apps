#!/usr/bin/env bash
# Day-to-day control: up | down | status | logs
#
# You should rarely need `up` -- both units autostart on boot. This exists for
# the gaming case (freeing the ~60 MB and stopping the polling) and for
# checking on things after a change.
set -euo pipefail

UNITS=(beszel-hub.service beszel-agent.service)

case "${1:-status}" in
  up)
    systemctl --user start "${UNITS[@]}"
    echo -n "waiting for hub"
    for _ in $(seq 1 60); do
      if curl -sf --max-time 2 http://127.0.0.1:8090/api/health >/dev/null; then
        echo " -- ready"
        # The hub answering says nothing about the agent: the hub is happy to
        # serve an empty dashboard. Check the agent separately.
        if [[ "$(systemctl --user is-active beszel-agent.service)" != "active" ]]; then
          echo "WARNING: hub is up but the agent is NOT -- the dashboard will be empty." >&2
          echo "         journalctl --user -u beszel-agent" >&2
          exit 1
        fi
        exit 0
      fi
      echo -n "."; sleep 1
    done
    echo; echo "WARNING: hub not healthy after 60 s -- journalctl --user -u beszel-hub" >&2
    exit 1
    ;;
  down)
    # Stopping the agent first means the hub never records the gap as a
    # "system down" event, which would otherwise sit in the history.
    systemctl --user stop beszel-agent.service beszel-hub.service
    echo "stopped"
    ;;
  logs)
    journalctl --user -u beszel-hub -u beszel-agent -n "${2:-50}" --no-pager
    ;;
  status)
    for u in "${UNITS[@]}"; do
      printf '%-24s %-10s restarts=%s\n' "$u" \
        "$(systemctl --user is-active "$u")" \
        "$(systemctl --user show -p NRestarts --value "$u")"
    done
    printf '%-24s %s\n' "podman.socket" "$(systemctl --user is-active podman.socket)"
    printf '%-24s %s\n' "hub /api/health" \
      "$(curl -sf --max-time 2 http://127.0.0.1:8090/api/health >/dev/null && echo ok || echo UNREACHABLE)"
    ;;
  *)
    echo "usage: $(basename "$0") {up|down|status|logs [n]}" >&2
    exit 2
    ;;
esac

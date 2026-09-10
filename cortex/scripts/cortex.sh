#!/usr/bin/env bash
# Start, stop and inspect the cortex stack.
#
#   cortex.sh status     what is running, and what it costs
#   cortex.sh up         start everything
#   cortex.sh down       stop everything (frees RAM and VRAM)
#   cortex.sh unload     free VRAM only, leave the UI usable
#
# No sudo anywhere. ollama.service is a system unit, but polkit allows the
# active local session to start and stop it, so this works as your own user.
set -euo pipefail

OLLAMA_API="http://127.0.0.1:11434"
UNITS=(open-webui searxng)

vram()  { nvidia-smi --query-gpu=memory.used,memory.total --format=csv,noheader 2>/dev/null || echo "n/a"; }
ram()   { free -m | awk '/^Mem:/{printf "%s MB used, %s MB available", $3, $7}'; }

loaded_models() {
  curl -sf "$OLLAMA_API/api/ps" 2>/dev/null | jq -r '.models[]?.name' 2>/dev/null || true
}

unload_models() {
  # Ollama reserves the full KV cache at load time, so an idle model still holds
  # its VRAM until keep_alive expires. This asks for it back now.
  #
  # Only ever called for models /api/ps says are loaded: a keep_alive:0 request
  # against a model that is NOT loaded will load it first, which is the exact
  # opposite of the point.
  local any=0
  while read -r m; do
    [[ -n "$m" ]] || continue
    any=1
    echo "  unloading $m"
    curl -sf "$OLLAMA_API/api/generate" \
      -d "$(jq -nc --arg m "$m" '{model:$m, keep_alive:0}')" >/dev/null || true
  done < <(loaded_models)

  [[ "$any" -eq 1 ]] || { echo "  nothing loaded"; return 0; }

  # Unloading is not instant. Measured at up to ~35 s -- the model sits in
  # "Stopping..." while the runner tears down and the driver reclaims VRAM.
  # Poll rather than sleeping a guessed interval.
  echo -n "  waiting for VRAM"
  for _ in $(seq 1 60); do
    [[ -z "$(loaded_models)" ]] && { echo " -- done"; return 0; }
    echo -n "."
    sleep 2
  done
  echo
  echo "  WARNING: still loaded after 120 s -- check 'ollama ps'" >&2
}

case "${1:-status}" in
  up)
    systemctl start ollama
    systemctl --user start "${UNITS[@]}"
    echo "Waiting for Open WebUI..."
    for _ in $(seq 1 60); do
      curl -sf -o /dev/null "http://127.0.0.1:8081/health" && break
      sleep 2
    done
    echo
    exec "$0" status
    ;;

  down)
    unload_models
    systemctl --user stop "${UNITS[@]}" || true
    systemctl stop ollama || true
    sleep 2
    echo "Stopped. RAM: $(ram). VRAM: $(vram)"
    ;;

  unload)
    # For gaming: VRAM is the contended resource, not RAM. This frees the ~3.2 GB
    # the model holds while leaving the UI up, so you can come back to your chats
    # without restarting anything. The next prompt just reloads the model.
    unload_models
    echo "VRAM: $(vram)"
    ;;

  status)
    printf 'ollama          %s\n' "$(systemctl is-active ollama)"
    for u in "${UNITS[@]}"; do
      printf '%-15s %s (restarts since boot: %s)\n' "$u" \
        "$(systemctl --user is-active "$u")" \
        "$(systemctl --user show "$u" -p NRestarts --value)"
    done
    echo
    echo "RAM:  $(ram)"
    echo "VRAM: $(vram)"
    echo
    if curl -sf -o /dev/null "$OLLAMA_API/api/tags" 2>/dev/null; then
      loaded=$(ollama ps 2>/dev/null | tail -n +2)
      echo "Models loaded: ${loaded:-none (VRAM is free)}"
    else
      echo "Models loaded: ollama is not running"
    fi
    echo
    printf 'Open WebUI     http://127.0.0.1:8081  |  https://cachyos.tail4f0f0b.ts.net:8443\n'
    ;;

  *)
    echo "usage: $(basename "$0") {status|up|down|unload}" >&2
    exit 2
    ;;
esac

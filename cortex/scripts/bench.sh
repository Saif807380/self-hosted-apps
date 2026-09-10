#!/usr/bin/env bash
# Measure generation speed and GPU/CPU placement for each model at each context size.
# A PROCESSOR reading of anything other than "100% GPU" means the model did not fit.
set -euo pipefail

HOST="${OLLAMA_HOST:-127.0.0.1:11434}"
PROMPT="Explain what a Kalman filter does, in one paragraph."

CTXS=(8192 16384 32768 65536)
MODELS=("$@")
if [[ ${#MODELS[@]} -eq 0 ]]; then
  MODELS=(qwen3-vl:4b qwen3-vl:2b)
fi

unload() {
  curl -sf "http://$HOST/api/generate" \
    -d "$(jq -nc --arg m "$1" '{model:$m, keep_alive:0}')" >/dev/null 2>&1 || true
}

printf '%-14s %-7s %9s %11s   %s\n' MODEL CTX TOK/S PREFILL PLACEMENT
printf '%s\n' "-------------------------------------------------------------------"

for model in "${MODELS[@]}"; do
  for ctx in "${CTXS[@]}"; do
    payload=$(jq -nc --arg m "$model" --arg p "$PROMPT" --argjson c "$ctx" \
      '{model:$m, prompt:$p, stream:false,
        options:{num_ctx:$c, num_predict:120, temperature:0}}')

    if ! response=$(curl -sf --max-time 600 "http://$HOST/api/generate" -d "$payload"); then
      printf '%-14s %-7s %9s\n' "$model" "$ctx" "FAILED"
      unload "$model"
      continue
    fi

    read -r tps prefill <<<"$(jq -r '
      [ (if (.eval_duration // 0) > 0
          then .eval_count / (.eval_duration / 1000000000)
          else 0 end),
        ((.prompt_eval_duration // 0) / 1000000) ] | @tsv' <<<"$response")"

    placement=$(ollama ps 2>/dev/null | awk -v m="$model" '$1 == m { $1=""; $2=""; print }' | xargs || true)

    printf '%-14s %-7s %9.1f %8.0f ms   %s\n' \
      "$model" "$ctx" "$tps" "$prefill" "${placement:-unloaded}"

    unload "$model"
  done
done

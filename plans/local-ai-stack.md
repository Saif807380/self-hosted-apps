# Local AI Stack — Plan

Working name: **cortex** (new top-level dir alongside `streamcloud`, `pitwall`, `trove`). Easy to rename before Phase 1.

## 1. Goals

From the brief, reordered by the stated priority (local-first > unified UI > offline), with **minimising cost** as the standing constraint:

1. Run a capable-enough model locally on this laptop.
2. Reach it from phone and tablet.
3. A real GUI — dashboard + chat.
4. Easy to set up, free.
5. Route to Claude/Gemini when the local model isn't enough.
6. Survives reboots, clearly documented.
7. Image input, web search, decent tooling.

## 2. The hardware reality

| Component | Spec | Verdict |
|---|---|---|
| CPU | i7-12700H, 14C/20T | Fine. Not the bottleneck. |
| RAM | 15 GiB total, **7.8 GiB already used, 5.7 GiB in zram swap** | **The real constraint.** |
| GPU | RTX 3050 Ti Mobile, **4096 MiB VRAM** (~430 MiB held by desktop) | Hard ceiling on model size. |
| Disk | btrfs, 199 GiB free | Plenty. |
| OS | CachyOS (Arch), podman 6.1.1, no docker | Rolling — favours fewer moving parts. |

Two things follow, and they drive every decision below.

**VRAM caps model size at ~3.6 GB.** That means 4B-class models at Q4. Not 8B, not 14B — those spill to CPU and crawl.

**System RAM is what will actually make this feel broken.** There is already 5.7 GiB in zram swap before adding anything. Open WebUI's *default* config loads a sentence-transformers embedding model into its own process (~500 MB+ per worker) on top of a ~1–2 GB baseline. Untuned, this stack lands on an already-swapping system and thrashes — and it'll look like the model is bad when it's actually memory pressure. Section 5 budgets this explicitly.

## 3. Honest expectations

You picked **images/OCR**, **private/sensitive text**, and **everyday chat & drafting** — and explicitly *not* coding.

That's genuinely good news. Those three are exactly where 4B models hold up, and they're the use cases where "good enough + private + free" beats "excellent + paid". A local 4B model will handle screenshot OCR, describing images, summarising and rewriting, and answering questions over your own documents at a quality you'll actually accept.

What it will *not* do: hard multi-step reasoning, long-context analysis, or reliable multi-tool agent loops. Had coding been on the list I'd have pushed back much harder.

On "move away from paid to free" — that's achievable for these three use cases, but be aware of one thing: **Anthropic has no free API tier, and your Claude subscription does not grant API access.** Routing to Claude from this stack means pay-per-token, separate from what you already pay. Gemini via Google AI Studio *does* have a real free tier, so the cost-conscious version of requirement #5 is "local first, Gemini free tier as the escape hatch, Claude only if you decide it's worth paying."

## 4. Architecture

```
  phone / tablet
        │  Tailscale (already up: cachyos.tail4f0f0b.ts.net)
        ▼
  tailscale serve --https=8443  ──► Open WebUI  :8081   [Quadlet container]
                                        │
                        ┌───────────────┼────────────────┐
                        ▼               ▼                ▼
                  Ollama :11434    SearXNG :8888    Gemini free tier
                  [native systemd]  [Quadlet]       [direct connection]
```

**Ollama runs native, not containerised.** You chose this and it's right: `ollama-cuda` from the Arch repos is a single static Go binary talking straight to the driver. Containerising it would add a CDI GPU-passthrough surface that needs `/etc/cdi/nvidia.yaml` regenerated after every NVIDIA driver bump — a recurring breakage on a rolling distro, bought for zero isolation benefit. The UI layer, which genuinely benefits from packaging, goes in containers.

**Open WebUI and SearXNG run as Podman Quadlets** in `~/.config/containers/systemd/`, using `network_mode: host` to match your existing stacks (and so the rootless container can reach Ollama on `127.0.0.1:11434`). Linger is already enabled, so user units survive logout.

**SearXNG** is a self-hosted metasearch engine — it queries Google/Bing/DuckDuckGo on your behalf, aggregates the results, strips the tracking, and hands them back over a local API. It's how requirement #7's "web support" gets met: Open WebUI calls it, gets live results, and feeds them to the model so it can answer about things past its training cutoff. The alternative is a paid search API (Brave, Serper, Tavily); SearXNG is the free, no-account, no-API-key option, which is what a cost-minimizing plan wants. It costs ~150 MB RAM. Droppable if you'd rather let Gemini's built-in search grounding handle current-events questions — but it's the only option here that doesn't hand a third party your query log.

**No LiteLLM.** My first instinct was a LiteLLM proxy for the routing layer, but Open WebUI speaks to OpenAI-compatible endpoints directly, and Gemini exposes one. Adding LiteLLM would cost another ~250 MB of RAM you don't have, to solve a problem you don't have yet. Revisit only if you want *other* clients sharing one routing endpoint (Section 8).

### Ports

| Service | Port | Notes |
|---|---|---|
| Ollama | 11434 | free |
| Open WebUI | 8081 | free |
| SearXNG | 8888 | free |
| Tailscale serve | 8443 | 443 is taken by Navidrome at `/` |

Verified free: 8081, 8443, 8888, 11434. In use on this box: 3000 (trove nginx), 4533 (Navidrome), 5432, 6379, 6767, 7878, 8080 (qBittorrent), 8096 (Jellyfin), 8191, 8384, 8989, 9696.

Open WebUI misbehaves under a path prefix, so `--set-path /ai` on the existing 443 listener is out; a second HTTPS port is the clean answer. Result: **`https://cachyos.tail4f0f0b.ts.net:8443`**. (Tailscale restricts serve to a small set of HTTPS ports; 8443 is expected to be valid but the docs don't enumerate them, so this gets verified with one command in Phase 2 rather than assumed.)

**This is not a second Tailscale server.** It's the same daemon and the same node you already run — `tailscale serve` is just config on it, adding a listener alongside the existing `443 → 4533` Navidrome mapping.

Worth being explicit about why serve is used at all, since Sonarr/Radarr don't need it: those are reached at plain `http://cachyos:8989` over the tailnet, and the same would work here at `http://cachyos:8081`. The difference is that browsers gate certain APIs behind a **secure context** (real HTTPS). Over plain HTTP, a phone will refuse camera access, microphone, and PWA install. Camera is directly relevant here — point the phone at a document and send it straight to the vision model — so the Let's Encrypt cert that serve provides is worth the extra port.

## 5. Budgets

### VRAM (4096 MiB total)

| Item | Size |
|---|---|
| Desktop compositor (kwin, steam overlay) | ~430 MiB |
| **Usable** | **~3.6 GB** |

Sizes below are the real `ollama` library figures, not estimates — I checked, and my initial recall was wrong by ~500 MB, which is exactly the margin that decides whether this fits.

| Model | Size | Vision | Fit |
|---|---|---|---|
| `qwen3-vl:4b` (q4_K_M) | **3.3 GB** | yes | Very tight. ~300 MB for KV cache. Will need `num_ctx` capped and q8_0 KV cache. |
| `qwen3-vl:2b` (q4_K_M) | **1.9 GB** | yes | Comfortable. Leaves room for the embedding model to stay resident. |
| `qwen3:4b` (q4_K_M) | **2.5 GB** | no | Comfortable, text-only, strong tool calling. |
| `qwen3:8b` (q4_K_M) | 5.2 GB | no | Does not fit. Spills to CPU. Not recommended here. |
| `nomic-embed-text` | ~275 MB | — | For RAG. |

### Context window — the cost nobody expects

Model weights are only half the VRAM story. The KV cache is the other half, and it scales linearly with context length.

Verified from the model configs: **Qwen3-VL-4B** is 36 layers / 8 KV heads / 128 head-dim, and **Qwen3-VL-2B** is 28 / 8 / 128. That gives a per-token KV cost of:

| Model | fp16 | q8_0 | q4_0 |
|---|---|---|---|
| Qwen3-VL-4B | 144 KiB/tok | 72 KiB/tok | 36 KiB/tok |
| Qwen3-VL-2B | 112 KiB/tok | 56 KiB/tok | 28 KiB/tok |

**The critical detail: llama.cpp preallocates the entire KV cache at model load.** Setting a 64K context does not cost more only when you use it — the full reservation is taken immediately, even for a two-line chat. This is the single most counter-intuitive thing about sizing a local model, and it's why the context number has to be chosen deliberately rather than left at the model's 256K default.

What that means in practice, against ~3.6 GB usable:

| Config | Weights | KV | Total | Verdict |
|---|---|---|---|---|
| 4B @ 8K, q4_0 KV | 3.3 GB | 0.3 GB | 3.6 GB | exactly at the line |
| 4B @ 16K, q4_0 KV | 3.3 GB | 0.6 GB | 3.9 GB | a few layers offloaded |
| 4B @ 64K, q4_0 KV | 3.3 GB | 2.4 GB | 5.7 GB | ~2.1 GB in system RAM |
| 4B @ 64K, q8_0 KV | 3.3 GB | 4.8 GB | 8.1 GB | far over |
| 2B @ 32K, q4_0 KV | 1.9 GB | 0.9 GB | 2.8 GB | comfortable |
| 2B @ 48K, q4_0 KV | 1.9 GB | 1.4 GB | 3.3 GB | fits |

**64K fully on GPU is not available on 4 GB with either model.** It is available with CPU offload, at a speed cost quantified below.

### Two profiles, not one choice

> **Superseded by the Phase 1 results in §6.** This section was written before measurement. The `long` profile does not work: at 39K tokens of real context the 4B generates at 1.6 tok/s, because the speeds below were all measured against an empty KV cache. Do not implement this table — see "The benchmark above measures the best case" under Phase 1. The committed config is a single profile at `num_ctx=16384`.

Rather than picking a single compromise, register both in Open WebUI. Ollama accepts `num_ctx` per request, so an unused profile costs nothing.

| Profile | Config | Speed | For |
|---|---|---|---|
| **fast** | 4B @ 16K, all GPU | snappy | 95% of it — chat, OCR, drafting |
| **long** | 4B @ 64K, partial offload | ~6–10 tok/s | pasting in a whole document |

Calibration on what actually consumes context: a screenshot is ~1–2K tokens, a long conversation 5–10K, retrieved RAG chunks 2–5K. 16K is roomier than it sounds.

### On spilling into system RAM

Layers that don't fit run on CPU against system RAM. Single-stream inference is memory-bandwidth-bound, so the ratio is what matters: this GPU does ~192 GB/s, system RAM ~51 GB/s (DDR4-3200) or ~77 GB/s (DDR5-4800). CPU-resident layers therefore run roughly **2.5–4× slower** than GPU-resident ones.

That's a slope, not a cliff — a 10–20% spillover costs perhaps 30–50% of throughput. Two caveats specific to this machine:

- The spilled weights land in RAM that is **already 5.7 GiB into zram swap**. If spillover pushes model data into swap, that *is* a cliff: per-token zram decompression is catastrophic. The `long` profile's ~2.1 GB of offload needs monitoring against free RAM.
- Prefill degrades far worse than generation, and images are token-heavy — so heavy offload hurts time-to-first-token most, which is the OCR path.

### Remaining levers

- `OLLAMA_FLASH_ATTENTION=1` and `OLLAMA_KV_CACHE_TYPE=q4_0` — the tables above already assume these.
- Pin the embedding model to CPU; it's tiny and latency-insensitive.
- Fall back to `qwen3-vl:2b`, which buys roughly double the context at the same footprint.

*Not doing:* moving the desktop to the Intel iGPU. It would free the ~430 MiB the compositor holds, but it's been ruled out, so every figure here assumes ~3.6 GB usable rather than ~3.9 GB.

Phase 1 measures rather than guesses — `ollama ps` reports the GPU/CPU split, and any CPU share means it didn't fully fit.

### System RAM (added load)

| Component | Untuned | Tuned |
|---|---|---|
| Open WebUI | 1.5–2 GB | ~600–800 MB |
| SearXNG | ~250 MB | ~150 MB |
| Ollama daemon (host side) | ~150 MB | ~150 MB |
| **Total added** | **~2.4 GB** | **~1.1 GB** |

Against ~7.5 GiB currently available, the tuned figure is comfortable and the untuned figure is not, given the existing swap. The tuning is all environment variables, verified against Open WebUI's own performance docs:

```
RAG_EMBEDDING_ENGINE=ollama          # biggest win — no sentence-transformers in-process
RAG_EMBEDDING_MODEL=nomic-embed-text
UVICORN_WORKERS=1                    # each worker replicates models
AUDIO_STT_ENGINE=webapi              # no local Whisper
ENABLE_IMAGE_GENERATION=False
ENABLE_CODE_INTERPRETER=False
ENABLE_AUTOCOMPLETE_GENERATION=False # fires per keystroke
ENABLE_FOLLOW_UP_GENERATION=False
ENABLE_TAGS_GENERATION=False
ENABLE_TITLE_GENERATION=False
```

The last four matter more here than they would on a bigger box. They're *task model* calls — Open WebUI generates chat titles, tags and follow-ups against whatever model is selected. With one 3.3 GB model resident and `OLLAMA_MAX_LOADED_MODELS` constrained, every new chat would fire an extra generation that contends with the reply you're waiting on. Disable them, or point the task model at `qwen3-vl:2b`.

If pressure still shows up, the levers are the natively-running media services (Jellyfin, Sonarr, Radarr, Prowlarr) — but that's your call, not something I'd change unasked.

## 6. Phases

Per `CLAUDE.md`, these ship one at a time with approval between each.

### Phase 1 — Local core
1. Create `cortex/` with the repo's existing layout conventions.
2. `pacman -S ollama-cuda` — confirmed present as `cachyos-extra-v3/ollama-cuda 0.33.2-1.1` (the CachyOS x86-64-v3 build). Ships `ollama.service` running as the `ollama` system user with state under `/var/lib/ollama`.
3. Env vars via drop-in at `/etc/systemd/system/ollama.service.d/override.conf` — never by editing the shipped unit, which package updates overwrite. Confirm the exact models path from `systemctl cat ollama.service` first.
4. `chattr +C` on the models directory **before** any pull — btrfs CoW fragments multi-GB model files badly, and the flag only affects files created after it's set.
5. Pull `qwen3-vl:4b`, `qwen3-vl:2b`, `nomic-embed-text`.
6. **Measure**: tok/s and GPU/CPU split per candidate (`ollama ps` shows the split; any CPU share means it didn't fit). Measure the `fast` (16K) and `long` (64K) profiles separately — the second one is the offload case and its speed needs to be a known number, not a hope.
7. Verify vision works — feed it a screenshot, check the OCR.

*Done when:* a model answers over `curl` at an acceptable speed, fully on GPU.

### Phase 1 results — measured 2026-09-10

Hardware after reboot: 3.7 GiB VRAM total / 3.5 GiB available, swap cleared to ~0. Ollama 0.33.3, `q8_0` KV cache, flash attention on.

| Model | ctx | tok/s | prefill | footprint | placement |
|---|---|---|---|---|---|
| qwen3-vl:4b | 8192 | 30.5 | 380 ms | 4.5 GB | 53%/47% CPU/GPU |
| qwen3-vl:4b | 16384 | 27.8 | 347 ms | 5.2 GB | 61%/39% |
| qwen3-vl:4b | 32768 | 25.9 | 405 ms | 6.6 GB | 67%/33% |
| qwen3-vl:4b | 65536 | 23.9 | 484 ms | 9.2 GB | 78%/22% |
| qwen3-vl:2b | 8192 | 121.9 | 47 ms | 2.0 GB | **100% GPU** |
| qwen3-vl:2b | 16384 | 89.6 | 143 ms | 3.5 GB | 38%/62% |
| qwen3-vl:2b | 32768 | 70.5 | 135 ms | 4.4 GB | 52%/48% |
| qwen3-vl:2b | 65536 | 59.4 | 200 ms | 6.5 GB | 69%/31% |

**Three predictions in this plan were wrong, and the corrections matter:**

1. **The 4B never fits on GPU, at any context.** Its real footprint at 8K is 4.5 GB, not the 3.3 GB of weights — the vision encoder and compute buffers were unaccounted for. Only `2b @ 8K` runs fully on GPU.
2. **Offload costs far less than estimated.** The §5 figure of ~6–10 tok/s for the offload case was badly pessimistic: the 4B holds 23.9–30.5 tok/s even at 78% CPU, and speed is nearly flat across context. The 14-core CPU absorbs it far better than the bandwidth ratio implied.
3. **Speed is therefore not the deciding factor.** Every configuration measured is well above reading speed, so the model choice comes down to quality.

**Quality test (invoice image, temperature 0, repeated).** Both models transcribed all text perfectly, including `KI-CAL-88231 / PO 4457-B`. Three questions, two runs each on the first:

| Question | Expected | 2b | 4b |
|---|---|---|---|
| Sum the line items, match the total? (item reads "(x3)") | 1,690.25 / yes | **wrong** both runs — 2,465.25 / no | correct both runs |
| Difference between Calibration and Freight | 1177.25 | 1177.25 ✓ | 1177.25 ✓ |
| TOTAL DUE less 10% | 1521.23 | 1521.225 ✓ | 1521.23 ✓ |

The 2B's **arithmetic is sound** — it gets both unambiguous calculations right. What it fails is **document semantics**: it reads "(x3)" as an instruction to multiply the line amount by three, rather than understanding that the amount shown is already the extended total. The failure is reproducible across runs, so it's a real behaviour rather than sampling noise. The 4B also rounds currency to cents where the 2B returns 1521.225.

This is narrower than "the 2B can't reason", and worth stating precisely: it is less reliable at interpreting document *conventions*, which is exactly the failure mode that matters for invoices, receipts and forms. That's enough to prefer the 4B for document work, but the 2B remains sound for transcription and straightforward calculation.

**Decision: `qwen3-vl:4b` as daily driver, `num_ctx=16384`, `q4_0` KV cache.** See the context-fill section below for why 16K rather than 32K or 64K — the short version is that the two-profile `fast`/`long` idea from §5 does not survive measurement, because long context is slow for reasons no configuration fixes. `qwen3-vl:2b` is retained as a fast transcription-only option at 122 tok/s, where its document-semantics weakness doesn't apply.

### The benchmark above measures the best case, not the real one

Every `tok/s` figure in the table was produced with a ~20-token prompt. That is generation against an essentially empty KV cache, and it is **not** what using the thing feels like. Measured with 39K tokens of actual context at 64K, `q4_0`:

| | empty-context bench | 39,438 tokens of real context |
|---|---|---|
| prefill | 484 ms | **75.4 s** (522 tok/s) |
| generation | 23.8 tok/s | **1.6 tok/s** |

Generation collapses **14×**. Each generated token must read the whole KV cache, and at 64K with 69% of it in system RAM, that read dominates everything else. A one-sentence answer over a long document costs ~75 s to first token and ~25 s more to finish.

**This kills the `long` profile as a usable idea.** The earlier claim that "64K works at 23.8 tok/s" was an artefact of benchmarking an empty context. Long-document work is not practical on this hardware — that is a real limitation to route to Gemini, not something to tune around.

The corollary matters just as much: the headline speeds hold for *short* exchanges and decay as a conversation fills.

### Context-fill curve — the number that should drive configuration

`qwen3-vl:4b`, `num_ctx=32768`, `q4_0`, varying the *actual* prompt length:

| real prompt tokens | prefill | generation | feel |
|---|---|---|---|
| 1,077 | 1.0 s | 22.5 tok/s | snappy |
| 2,224 | 2.0 s | 18.3 tok/s | snappy |
| 4,300 | 3.8 s | 13.8 tok/s | comfortable |
| 8,448 | 7.9 s | 9.9 tok/s | usable |
| 16,639 | 18.0 s | 5.5 tok/s | sluggish |
| 39,438 | 75.4 s | 1.6 tok/s | unusable |

Generation roughly halves for every doubling of filled context; prefill is near-linear at ~1 s per 1,000 tokens. **The practical ceiling is ~8K tokens of real content**, with 16K tolerable for occasional longer work.

Note that `num_ctx` and context *fill* do different damage: `num_ctx` fixes the up-front footprint and GPU/CPU split, while fill determines speed. So the two are tuned separately — and the allocation does help: at `num_ctx=16384` with an 8,448-token prompt the 4B gives **11.3 tok/s**, against 9.9 at `num_ctx=32768` for the same prompt.

### The 2B does not decay the same way, and that changes its role

Same test, `num_ctx=16384`:

| Model | 4,300 tok fill | 8,448 tok fill | footprint | placement |
|---|---|---|---|---|
| `qwen3-vl:4b` | 13.8 tok/s | 11.3 tok/s | 4.5 GB | 52%/48% CPU/GPU |
| `qwen3-vl:2b` | **98.9 tok/s** | **84.1 tok/s** | 2.1 GB | **100% GPU** |

The 2B holds ~7× the throughput at 8K of real context and barely decays, because at 16K its entire KV cache still fits in VRAM. This confirms the decay mechanism: the collapse is RAM-resident KV being re-read per token, not context length as such.

**Revised two-model split — by task, not by context length.** The original `fast`/`long` idea was the wrong axis. The right one:

| Use | Model | Why |
|---|---|---|
| Questions *about* documents, invoices, forms, anything where being wrong is costly | `qwen3-vl:4b` | Only model that reads document conventions correctly |
| Transcription/OCR, summarising long text, bulk drafting | `qwen3-vl:2b` | ~7× faster, stays on GPU, and OCR was pixel-identical to the 4B |

This also helps Phase 2: the 2B's 2.1 GB leaves both VRAM and system RAM free for Open WebUI and SearXNG in a way the 4B does not.

**Revised recommendation: `num_ctx=16384`.** Footprint 4.5 GB at 48% GPU, which leaves room for Phase 2's containers, and it comfortably holds the realistic case — a screenshot (~1–2K tokens) plus a question plus some history — inside the responsive part of the curve. Longer documents are a Gemini job, not a local one.

**Default context is 4096, and this is a Phase 2 trap.** A request that doesn't set `num_ctx` gets 4K — not the model's 256K, and not anything measured above (`OLLAMA_CONTEXT_LENGTH:0` means "use the built-in default"). Confirmed identical on `/api/generate` and `/api/chat`, the latter being what Open WebUI calls. So Open WebUI's per-model entry **must** set context explicitly, or long documents will be silently truncated at 4K with no error. Footprint at 4K is 4.1 GB, 48%/52% CPU/GPU.

**Open:** KV cache is currently `q8_0`. Moving to `q4_0` would roughly halve KV cost (32K: 2.4 GB → 1.2 GB), putting more on GPU and freeing RAM for Phase 2's containers. Needs a service restart, and the quality tests above must be re-run afterwards — the 4B's document-semantics advantage is the entire reason it was chosen, so it has to survive the quantization to be worth taking.

### Phase 2 — GUI and access
1. Open WebUI Quadlet with the tuned env above.
2. SearXNG Quadlet, wired into Open WebUI's web search. **`formats: [html, json]` must be added to SearXNG's `settings.yml`** — JSON output is off by default, and Open WebUI needs it. This is the most common failure in this exact wiring and it fails *silently*: web search just returns nothing.
3. `tailscale serve --bg --https=8443 8081` (verifying the port is permitted).
4. Install as a PWA on the Pixel and the Tab S10.
5. Reboot test.

*Done when:* you can open it on your phone, ask a question, and upload a screenshot.

### Phase 2 results — measured 2026-09-10

Five things about the plan above turned out to be wrong or incomplete.

**1. Context belongs in a Modelfile, not in Open WebUI.** The `num_ctx=4096`
default found in Phase 1 is best fixed server-side. `cortex/modelfiles/` builds
two tags — `cortex-4b` and `cortex-2b` — that carry `PARAMETER num_ctx 16384`,
so *every* client gets the right context without configuring anything. Verified:
`ollama ps` reports `CONTEXT 16384` for a request that sets no options at all.
The alternative, Open WebUI's per-model Advanced Params, would have buried the
single most important setting in an untracked SQLite DB.

They cost a manifest each, not 3.3 GB — `ollama create` reuses the existing
blobs.

**2. One `.env` was the wrong shape.** Podman hands an `--env-file` to the
container wholesale, so a single shared file would have put the Phase 3 Gemini
API key into SearXNG's environment — a service whose entire job is making
outbound requests to a few dozen third-party search engines. Split into
`cortex/.env.searxng` and `cortex/.env.open-webui`, both matching the root
`.gitignore`'s `.env.*` rule (verified with `git check-ignore` before any
secret was written into them), both mode 600.

**3. SearXNG's shipped image binds to every interface, and the variable that
guides say fixes it does nothing.** This image runs on **granian**, not uwsgi.
`SEARXNG_BIND_ADDRESS` and the `UWSGI_*` knobs are silently ignored; only
`SEARXNG_PORT` is aliased through. Combined with `Network=host` the first start
put SearXNG on `*:8888` — reachable from the LAN and the whole tailnet.
`GRANIAN_HOST=127.0.0.1` is the real knob. Now verified in both directions:
loopback answers, the tailnet IP is refused.

Worth re-checking after any image bump: `ss -ltnp | grep 8888` must show
`127.0.0.1`, not `*`.

**4. The secret can't be injected the documented way.** The entrypoint's
`sed` over `ultrasecretkey` only runs when `settings.yml` does *not* already
exist, and it needs the file writable — which would mean either dropping `:ro`
or letting a secret be written into a tracked repo file. Instead the tracked
`cortex/searxng/settings.yml` carries a `@SEARXNG_SECRET@` placeholder, and
`install-quadlets.sh` renders it to `~/.local/share/cortex/searxng/settings.yml`
at install time. Mount stays read-only, repo stays clean.

**5. `formats: [html, json]` was the one thing the plan got exactly right.**
Confirmed working: `?format=json` returns 200 with parsed results, and the
`limiter: false` setting means Open WebUI's queries aren't rate-limited into
silence.

**6. `CORS_ALLOW_ORIGIN` defaults to `*`**, which Open WebUI warns about on
every start. It takes a `;`-separated list, so both the tailnet origin and
loopback fit. Notably this one is *not* PersistentConfig — it is re-read from
the environment every start, so unlike the rest of the table it can be changed
later without touching the database.

#### Measured RAM — the number this phase turned on

| Component | Plan (untuned) | Plan (tuned) | **Measured** |
|---|---|---|---|
| Open WebUI | 1.5–2 GB | 600–800 MB | **711.7 MB** |
| SearXNG | ~250 MB | ~150 MB | **156.1 MB** |
| **Total added** | ~2.4 GB | ~1.1 GB | **867.8 MB** |

`/proc/pressure/memory` full avg10/60/300 all `0.00` with both containers warm,
8.3 GiB still available. The tuning worked: this landed under the tuned budget,
not near the untuned figure.

Verified the persisted config survived a restart rather than being re-seeded,
which is the failure this table would otherwise hide.

**7. `After=ollama.service` in a user unit does nothing.** These are user
units; `ollama.service` and `network-online.target` live in the system manager,
which the user manager cannot see. The line looks reassuring and has no effect.
`podman-user-wait-network-online.service` is the user-scope equivalent podman
ships. Ollama itself stays unorderable from here — `Restart=always` covers the
race instead.

This makes the reboot check `NRestarts`, not `is-active`: with `Restart=always`
a boot-time race self-heals and the end state looks identical to a clean start.

```
systemctl --user show open-webui searxng -p NRestarts   # 0 is the pass
```

#### Verified against the tag Open WebUI actually uses

`ollama create` preserves capabilities — `cortex-4b` reports `vision`, `tools`,
`thinking`, matching `qwen3-vl:4b` exactly — and an OCR test against
`cortex-4b` (not the base tag) read all three fields off a test invoice
correctly. Worth checking rather than assuming: had the vision projector not
carried across, Open WebUI would route uploads through the document pipeline
instead, and answer plausibly without ever looking at the image.

CORS was verified with an actual `Origin` header, not `curl /health` — a health
check sends no `Origin` and so never exercises CORS at all. Both configured
origins are echoed; an unlisted one is not.

**Sudo was never needed.** The plan flagged `tailscale serve` as the one likely
root step; the operator is already set to `saifkazi`, so all of Phase 2 ran
unprivileged. Port 8443 is accepted, and the Let's Encrypt certificate is valid
to Nov 25 2026 — closing the "Tailscale rejects 8443" risk in §8.

**Reboot survival** comes from `WantedBy=default.target` in each Quadlet's
`[Install]` section — without it the units generate fine and start fine by hand
but never come back. Verified indirectly by the generator creating
`default.target.wants/{open-webui,searxng}.service` symlinks, and directly by
the reboot test below.

### Model selection was wrong — the default tags are thinking variants

Phase 1 chose `qwen3-vl:4b` and `qwen3-vl:2b` without checking what those tags
resolve to. Both manifests say:

```
"renderer": "qwen3-vl-thinking",  "parser": "qwen3-vl-thinking"
```

They are the **thinking** variants. For the stated use cases — OCR, screenshots,
private text, everyday chat and drafting, explicitly not coding — reasoning is
pure overhead, and on a 2B it does not converge.

This surfaced as a user-visible failure: a web-searched weather question that
retrieved its sources correctly and then returned nothing at all.

Same prompt, same 151-token context, cold GPU, `num_ctx=16384`:

| | `cortex-2b` (thinking) | `qwen3-vl:2b-instruct` |
|---|---|---|
| wall time | **210 s** | **4 s** |
| tokens generated | 16,233 | 63 |
| `done_reason` | **`length`** | `stop` |
| thinking output | 41,865 chars | 0 |
| visible answer | **empty** | correct, 2 lines |

`done_reason=length` is the whole story: it exhausted the generation limit
mid-reasoning and never reached an answer. Not slow — *non-terminating*. The
78 tok/s it sustained is irrelevant when none of those tokens reach the user.

`think: false` does not fix it. The behaviour is in the model's own template,
and the Modelfile template is `{{ .Prompt }}` — there is nothing to override.
The variant has to change.

`qwen3-vl:2b-instruct` keeps `vision` and `tools`; only `thinking` is gone.

**Superseding the Phase 1 recommendation: use the `-instruct` variants.** The
task split (4B for documents, 2B for OCR and bulk) still holds — it is the
thinking/instruct axis that was wrong, not the size axis.

### Phase 3 — Free cloud escape hatch
1. Google AI Studio key (free tier, no card) → `.env`.
2. Add Gemini as an Open WebUI direct connection, alongside the local model in the same dropdown.
3. Optionally OpenRouter's `:free` models as a secondary.
4. Document which work goes where.

*Done when:* one dropdown, local default, cloud one click away. **No API keys are committed** — `.env` gitignored, mirroring `streamcloud/.env`.

### Phase 4 — Docs and hardening
1. `cortex/README.md` in the style of `streamcloud/README.md`: architecture, restart runbook, tuning knobs, troubleshooting.
2. Model-swap and update procedure.
3. Optional: MCP tools via `mcpo`. Flagged as optional because 4B models are mediocre at multi-tool orchestration — the pragmatic answer is routing tool-heavy work to Gemini.

## 7. Extending it later

Nothing here is a dead end — Open WebUI is the extension point, and all of this is additive after Phase 4:

- **Tools** — Python functions written in the UI or dropped in as files; any tool-calling model can invoke them.
- **MCP servers** via `mcpo`, Open WebUI's MCP↔OpenAPI proxy, so anything already built as an MCP server plugs straight in.
- **Functions / Pipelines** — filters and custom backends for pre- and post-processing.
- **Models** — saved presets pairing a system prompt, a set of tools and a knowledge base. This is the closest analogue to a "skill".
- **Knowledge** — RAG collections attached per model.

The limiting factor is model capability, not the platform. 4B models are mediocre at multi-step tool orchestration, and `qwen3:4b` (text) is meaningfully better at tool calling than the VL variants. The realistic pattern is simple single-tool calls locally, with tool-heavy work routed to Gemini.

## 8. Risks

| Risk | Mitigation |
|---|---|
| `qwen3-vl:4b` doesn't fit in 3.6 GB | Measured in Phase 1; fallbacks are `qwen3-vl:2b` or a lower `num_ctx` (iGPU offload is ruled out). |
| `long` profile's 2.1 GB offload pushes the box into swap | Measured in Phase 1, not assumed; drop `long` to 32K if free RAM won't carry it. |
| RAM thrashing on an already-swapping box | Tuned env from the start; budget table above. |
| 4B quality disappoints | Expectations set in §3; Gemini free tier as the hatch. |
| NVIDIA driver bump breaks inference | Native ollama avoids the CDI surface entirely. |
| Tailscale rejects port 8443 | Verified in Phase 2; fallbacks are 10000 or a dedicated Tailscale Service hostname. |

## 9. Deferred

- **LiteLLM** — only if other clients need one shared routing endpoint.
- **Whisper / TTS** — no RAM budget for it today.
- **Image generation** — 3.6 GB VRAM doesn't have room alongside an LLM.
- **Moving to a dedicated host** — this plan assumes the laptop; nothing here blocks a later move.

## 10. Branch contents

`CLAUDE.md` and `docs/agents/` (from the skills setup) plus the modified `streamcloud/README.md` ride along on this branch rather than being split onto `main` first.

# cortex

A local AI stack on this laptop: a vision-capable model you own, a chat UI that
works from the phone, and private web search. Free, and nothing leaves the
tailnet unless you deliberately send it out.

Planned in `plans/local-ai-stack.md`, which also carries every measurement
quoted below.

---

## Where it lives

| What | URL | Notes |
|---|---|---|
| Open WebUI | `https://cachyos.tail4f0f0b.ts.net:8443` | **Use this one.** From any device on the tailnet |
| Open WebUI (local) | `http://127.0.0.1:8081` | Same thing, laptop only |
| Ollama API | `http://127.0.0.1:11434` | For scripts. No UI |
| SearXNG | `http://127.0.0.1:8888` | Loopback only, on purpose. Open WebUI talks to it; you don't |

Use the HTTPS tailnet URL on the phone, not `http://cachyos:8081`. Both work,
but browsers only allow **camera, microphone and "Add to Home Screen"** on a
real HTTPS origin. Pointing the camera at a document is the whole point, so
take the certificate.

---

## Which model to use

Two models, and the split is **by task, not by document length**.

| Task | Model | Why |
|---|---|---|
| Questions *about* a document — invoices, forms, letters, anything where being wrong costs you | **cortex-4b** | The only one that reads document conventions correctly |
| OCR, transcription, summarising, bulk drafting, quick chat | **cortex-2b** | ~7× faster, and its OCR was pixel-identical to the 4B |

Both see images. Both are set to a 16K context. `cortex-2b` also runs chat
titles and tags in the background, so it stays warm.

### The honest speed picture

`cortex-4b` doesn't fit in 4 GB of VRAM — about half of it runs on CPU. That's
fine for short exchanges and degrades as a conversation grows, because every
generated token re-reads the whole conversation:

| Conversation size | cortex-4b | cortex-2b |
|---|---|---|
| ~1,000 tokens | 22.5 tok/s | — |
| ~4,300 tokens | 13.8 tok/s | 98.9 tok/s |
| ~8,400 tokens | 11.3 tok/s | 84.1 tok/s |
| ~16,600 tokens | 5.5 tok/s | — |
| ~39,000 tokens | **1.6 tok/s** | — |

`cortex-2b` barely decays because it fits entirely in VRAM. `cortex-4b` roughly
halves for every doubling of conversation length.

**Practical rule: start a new chat often.** A long thread gets slow because of
its own history, not because of the question you just asked. A screenshot plus
a question is ~1–2K tokens and sits in the fast part of that curve.

**What to send elsewhere.** Long-document analysis is genuinely not practical
here — a 39K-token document takes 75 seconds before the first word appears.
That's a Gemini job (Phase 3), not something to tune around. Same for hard
multi-step reasoning and anything involving code.

---

## Using it day to day

**Pick the model** from the dropdown at the top of the chat. It's per-chat, and
changing it mid-conversation is fine.

**Images** — drag one in, or use the `+` button. On the phone, `+` offers the
camera directly. Ask your question in the same message as the image.

**Web search** — toggle it on beneath the message box before sending. Working
looks like **sources appearing under the reply**. If no sources appear, it
failed silently; see troubleshooting.

Expect the first web-searched question to be noticeably slower than a plain
one. It runs three models — `nomic-embed-text` to embed results, your chat
model, and `cortex-2b` for the title — against a two-model limit and 4 GB of
VRAM. Ollama evicts and reloads rather than failing.

**Documents** — attach with `+` for one-off questions, or build a Knowledge
collection in Workspace for something you'll ask about repeatedly. Remember the
speed table: a large document will be slow whichever way it arrives.

**Admin settings** are under your avatar → Settings → Admin. Signup is open by
default; new accounts land in `pending` and can do nothing until you approve
them, but turning signup off entirely is the tidier answer once your own
account exists.

---

## Starting and stopping

One script does everything. **No sudo** — `ollama.service` is a system unit, but
polkit lets your local session control it.

```bash
cortex/scripts/cortex.sh status     # what's running, and what it costs
cortex/scripts/cortex.sh up         # start everything (~22 s to healthy)
cortex/scripts/cortex.sh down       # stop everything (~10 s)
cortex/scripts/cortex.sh unload     # free VRAM only, leave the UI up
```

### Before gaming

**VRAM is the contended resource, not RAM** — and the answer depends on how
much you want back:

| | RAM freed | VRAM freed | UI still up? |
|---|---|---|---|
| `unload` | ~0 | **3.2 GB** | Yes |
| `down` | ~1.4 GB | **3.2 GB** | No |

Measured: a loaded `cortex-4b` holds **3,284 MiB of the 4,096 MiB** on the card.
Unloaded, the GPU sits at 106 MiB.

`unload` waits for the VRAM to actually come back rather than returning
optimistically. Usually that's under a second; it has been measured at ~35 s,
where the model sits in `Stopping...` while the runner tears down. When it
returns, the memory is genuinely free — so it's safe to launch the game on the
next line.

**`unload` is usually what you want.** It gives back all the VRAM, keeps your
chats reachable, and the next prompt just reloads the model — no restart, no
lost state. Reach for `down` when you want the ~1.4 GB of RAM as well, or you're
done for the day.

You often need neither. `OLLAMA_KEEP_ALIVE=5m` means an idle model unloads
itself after five minutes. `unload` is for when you don't want to wait, and for
the case that actually bites: a model still resident from a chat ten minutes
ago, because the timer restarts on every message.

---

## After a reboot

**Nothing to do.** All three come back on their own — linger is enabled and each
Quadlet declares `WantedBy=default.target`.

To check, don't use `is-active`:

```bash
cortex/scripts/cortex.sh status
```

Look at **restarts since boot**. `0` is the pass.

Non-zero doesn't mean it's broken — it means something started before its
dependencies and `Restart=always` papered over it, and you'd never see that
from `is-active`. These are *user* units and can't order themselves against
`ollama.service`, which lives in the system manager. Consistently non-zero is
worth investigating; the plan's Phase 2 notes explain why the ordering is
shaped this way.

---

## Troubleshooting

### Web search returns no sources

The failure is silent by design — the reply still arrives, just without having
searched. In order of likelihood:

```bash
# 1. Is SearXNG up and answering JSON?
curl -s "http://127.0.0.1:8888/search?q=test&format=json" | head -c 200
```

Empty or an error means `formats: [html, json]` didn't survive. JSON output is
off in stock SearXNG and Open WebUI speaks only JSON — this is the single most
common failure in this wiring.

```bash
# 2. What did Open WebUI actually try?
journalctl --user -u open-webui | grep -i searxng | tail
```

### Replies are very slow

Check the conversation length first — see the speed table. Then:

```bash
ollama ps    # PROCESSOR column
```

`cortex-4b` showing `52%/48% CPU/GPU` is normal and expected. `cortex-2b`
showing anything other than `100% GPU` is not — something else is holding VRAM.

### It's unreachable from the phone

```bash
tailscale status | head -3          # is the tailnet up on both ends?
tailscale serve status              # should map :8443 -> 127.0.0.1:8081
cortex/scripts/cortex.sh status
```

If `serve status` lost the `:8443` mapping, `cortex/scripts/tailscale-serve.sh`
puts it back.

### A model answers as if it never saw the image

Check the model actually has vision:

```bash
ollama show cortex-4b | grep -iA4 capabilit    # want: vision
```

Without it, Open WebUI silently routes uploads through the document pipeline
and answers plausibly from filename and context alone.

### Changing a setting in the Quadlet does nothing

Most of Open WebUI's configuration is **PersistentConfig**: read from the
environment once on first boot, written to the database, and the environment
variable ignored from then on. Editing `open-webui.container` and restarting
will not change it — use the UI instead.

The exceptions, re-read every start, are `UVICORN_WORKERS`, `HOST`, `PORT`,
`CORS_ALLOW_ORIGIN` and the `WEBUI_*` security variables.

---

## Layout

```
cortex/
├── modelfiles/          cortex-4b, cortex-2b -- pins num_ctx to 16384
├── quadlets/            systemd container units (installed, not run from here)
├── searxng/             settings.yml template, @SEARXNG_SECRET@ placeholder
├── systemd/             ollama.service drop-in
├── scripts/
│   ├── cortex.sh              up / down / unload / status
│   ├── install-quadlets.sh    installs units, renders settings, sets +C
│   ├── tailscale-serve.sh     publishes :8443 on the tailnet
│   └── bench.sh               model speed and GPU/CPU placement
├── .env.searxng         secrets, gitignored, mode 600
└── .env.open-webui      secrets, gitignored, mode 600
```

Secrets are split per container deliberately: podman hands an `--env-file` to
the container whole, so one shared file would put the Gemini API key into
SearXNG's environment — a service that exists to make outbound requests to
dozens of third-party engines.

### Changing configuration

```bash
# 1. edit cortex/quadlets/*.container or cortex/searxng/settings.yml
cortex/scripts/install-quadlets.sh      # copies units, re-renders settings
systemctl --user restart open-webui searxng
```

The installed units under `~/.config/containers/systemd/` are **copies**.
Editing them directly works until the next install, which overwrites them.

### Changing a model's context

Edit the Modelfile, then rebuild — it reuses the existing weights, so this
costs a manifest, not another 3.3 GB:

```bash
ollama create cortex-4b -f cortex/modelfiles/cortex-4b.Modelfile
ollama show cortex-4b --parameters
```

Doing it here rather than in Open WebUI's Advanced Params means every client
gets it, and it lives in git. Ollama's own default is **4096** — a client that
sets nothing silently truncates at 4K.

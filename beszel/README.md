# beszel

Monitoring dashboard for everything on this laptop — streamcloud, trove,
cortex, and the machine itself. Historical CPU, memory, disk, network,
temperatures, GPU, per-container stats and per-service stats.

**Open it at <https://cachyos.tail4f0f0b.ts.net:8444>** from any device on the
tailnet. There is no login screen.

---

## What it does *not* do

**Beszel is read-only.** There is no restart button, no stop button, no start
button — not for containers, not for systemd services, not in 0.19.0 and not in
any earlier release. If a container is wedged, you still fix it from a terminal.

The one thing it reads beyond metrics is `CONTAINER_DETAILS`, which gives you
**container inspect and log viewing** in the UI. That is genuinely useful — it
covers most of what you'd otherwise open a terminal for — but it is still
reading, not acting.

If one-click restart is something you want, that is a second tool
(Portainer, Yacht, Dozzle for logs), not a Beszel setting.

---

## No authentication — and what that costs

You asked for no auth, and there is a clean way to get it. It is **not**
`DISABLE_PASSWORD_AUTH`, which is what most search results will tell you —
that one *forces OAuth*, which is the opposite. The right switch is:

    AUTO_LOGIN=<your email>

which authenticates every single request as that user. No login screen, no
session, no cookie. Straight to the dashboard, on the phone too.

**The consequence, stated plainly: anything that can reach the port is logged
in.** There is no second factor and no password prompt. That is why:

| | |
|---|---|
| the hub binds `127.0.0.1:8090` | not `0.0.0.0`, so nothing on the WiFi LAN can reach it |
| the agent binds `127.0.0.1:45876` | same |
| remote access is `tailscale serve` | tailnet membership is the *only* thing in front of it |

Verified after setup: `ss -tln` shows both on `127.0.0.1` and nothing else.

**Do not** change the hub's `Exec=serve --http=...` to `0.0.0.0`, and do not
add a `PublishPort`. With `AUTO_LOGIN` on, that publishes an unauthenticated
dashboard to every device on whatever network you are joined to.

One exception to "no login": the PocketBase superuser screens at `/_/`
(raw collections, backups) still prompt. `AUTO_LOGIN` covers the regular user,
not the superuser. That password is `USER_PASSWORD` in `.env.beszel-hub` —
generated at install, and break-glass only.

---

## Reading the dashboard per app

**Beszel has no groups and no tags.** One agent is one "System" row, and there
is no way to label a container as belonging to streamcloud. This was a real
fork in the design and it was decided deliberately:

The alternative was three agents — `streamcloud`, `trove`, `cortex` — each
scoped with `EXCLUDE_CONTAINERS`. That produces three rows with those names,
which is what you actually asked for. It was rejected because **all three
would report the same host**: identical CPU, memory, disk and network on every
row, three times over, because there is only one machine. And `EXCLUDE_CONTAINERS`
is exclusion-only, so every new app would have to be added to the other two
exclusion lists or it would leak into all three views.

So: one accurate system, and the split is done with the **filter box** in the
Containers and Services tabs. The names already carry their app:

| Type in the filter | You get |
|---|---|
| `compose_` | streamcloud's containers — gluetun, qbittorrent, flaresolverr |
| `infra_` | trove — postgres, redis, backend, ui |
| `open-webui` / `searxng` | cortex |
| `jellyfin`, `sonarr`, `radarr`, `prowlarr`, `bazarr`, `navidrome` | streamcloud's services (Services tab) |

If you later decide the duplicated host rows are worth it, the three-agent
version is a copy of `systemd-user/beszel-agent.service` per app with a
different `LISTEN` port, `EXCLUDE_CONTAINERS`, and a `config.yml` entry each.

---

## What is monitored

**Containers** — everything podman is running, whoever started it. Measured at
setup: all 7 running containers plus `beszel-hub` itself.

**Services** — only the ones named in `SERVICE_PATTERNS` in the agent unit.
Without that list the agent collects all ~160 system units. Currently:

    jellyfin  navidrome  sonarr  radarr  prowlarr  bazarr
    ollama  tailscaled  systemd-resolved  proton.VPN
    trove-tailscale-cert.timer

A service only appears once it has been active at least once this boot. So
`ollama` is **absent until you run `cortex/scripts/cortex.sh up`** — that is
expected, not a fault.

**The blind spot worth knowing:** the agent reads the **system** D-Bus only
(`agent/systemd.go` calls `dbus.NewSystemConnectionContext`). **User units are
invisible in the Services tab.** Cortex's Open WebUI and SearXNG are user
Quadlets, so they show up under Containers and never under Services. Same for
streamcloud's user timers (`generate-daily-playlists`, `lastfm-discovery`, …).

**GPU** — pinned to `nvidia-smi`, showing the RTX 3050 Ti's VRAM against its
4000 MiB. This is the number to watch when cortex is loaded: a model that spills
out of VRAM shows up here before it shows up as slow generation. The Intel Iris
Xe is not collected — `intel_gpu_top` needs root or `CAP_PERFMON`, and letting
auto-detection try it just half-fails.

**Temperatures** — four sensors: CPU package, GPU, NVMe, battery. The systems
table shows the CPU package (`PRIMARY_SENSOR`); without that pin it shows
whichever sensor is hottest, which flips between CPU and GPU mid-session.

---

## Starting and stopping

Both units **autostart on boot**. This is the opposite of what cortex does, on
purpose: a monitor that only runs when you remember to start it has no history,
and history is most of what you want from it. The cost is small —

| | Measured |
|---|---|
| `beszel-hub` | 20.4 MB RAM |
| `beszel-agent` | 29.4 MB RAM |
| VRAM | none |
| `/proc/pressure/memory` | flat at 0.00 |

About 50 MB total. Open WebUI alone was measured at 712 MB, so this is roughly
a fourteenth of it, and it holds no VRAM at all. There is no gaming case for
stopping it — but if you want to:

    beszel/scripts/beszel.sh down      # stop both
    beszel/scripts/beszel.sh up        # start both, wait for healthy
    beszel/scripts/beszel.sh status    # active? restart count? hub reachable?
    beszel/scripts/beszel.sh logs 100  # both units, interleaved

`down` stops the agent before the hub, so the hub never records the gap as a
"system down" event sitting in your history.

---

## After a reboot

**Nothing to do** — both units are set to come back on their own.

Stated precisely, because it matters: **this has not been through an actual
reboot yet.** What has been verified is the machinery, not the event. Next time
you restart, the honest check is the restart *count*, not `is-active` —
`Restart=always` will happily paper over a boot race and still show green:

    systemctl --user show -p NRestarts --value beszel-hub.service
    systemctl --user show -p NRestarts --value beszel-agent.service
    beszel/scripts/beszel.sh status

Non-zero restarts mean something lost a race at boot even though it looks fine
now. Same check, and the same reason, as cortex.

| Component | Autostart | Mechanism |
|---|---|---|
| `beszel-hub.service` | on | `[Install]` in the Quadlet → generator makes the `default.target.wants` symlink |
| `beszel-agent.service` | on | ordinary `systemctl --user enable` |
| `podman.socket` | on | `systemctl --user enable` — **required**, see below |
| tailscale serve `:8444` | on | `--bg` persists across reboots |

`podman.socket` is the one that is easy to lose. It is **not** enabled by
default on Arch, and Beszel reads container stats over the Docker Engine API
that this socket serves. Without it the Containers tab is simply empty — no
error, no warning, just nothing. `beszel.sh status` prints its state for
exactly this reason.

The boot race that would break this *was* tested, short of rebooting: stopping
`podman.socket` and then starting the agent brought the socket back up on its
own and containers collected normally. That is the agent unit's
`Wants=podman.socket` doing its job, with `After=` guaranteeing the ordering.
So the ordering is sound; it is the reboot itself that is unobserved.

Note `systemctl --user enable beszel-hub` **will not work** — Quadlet units are
generated into `/run` and systemd refuses to enable a generated unit. The
`[Install]` section inside `quadlets/beszel-hub.container` is what creates the
symlink, via the generator. Same trap as cortex.

---

## Reinstalling / changing config

    beszel/scripts/install.sh

Idempotent and safe to re-run. It regenerates nothing that already exists and
never touches the hub's database. **No sudo at any point.**

It will: install the agent binary if missing, enable `podman.socket`, create the
data directories with `chattr +C`, generate the two `.env` files if absent,
render `config.yml`, install both unit files, start the hub, read the hub's
public key out of `~/.local/share/beszel/hub/id_ed25519` into
`.env.beszel-agent`, and start the agent.

After editing a unit file, re-run it — or for the agent alone:

    install -m 644 beszel/systemd-user/beszel-agent.service ~/.config/systemd/user/
    systemctl --user daemon-reload && systemctl --user restart beszel-agent

**Updating:**

    beszel-agent update                                  # self-update, no root
    podman pull docker.io/henrygd/beszel:<new tag>       # then bump the Quadlet's Image=

The agent is in `~/.local/bin`, not from the AUR, deliberately: the AUR package
installs a *system* service running as a dedicated `beszel` user, and that user
cannot read `/run/user/1000/podman/podman.sock`. Every container on this box is
rootless under uid 1000, so the agent has to run as you.

---

## Troubleshooting

**Containers tab empty.** `systemctl --user is-active podman.socket`. If it is
dead, `systemctl --user enable --now podman.socket` and restart the agent.

**Services tab empty or missing something.** Check the unit is a *system* unit,
not a user unit — user units are never collected (see the blind spot above).
Then check it matches `SERVICE_PATTERNS`, and that it has been active at least
once since boot.

**System shows red / down.** `beszel.sh status` first. If the agent is active
but the hub says down, the key is out of sync — re-run `install.sh`, which
rewrites `.env.beszel-agent` from the hub's current key.

**Dashboard asks for a password.** You are on a `/_/` superuser screen.
`USER_PASSWORD` in `.env.beszel-hub`.

**...and that password is rejected.** `USER_EMAIL` / `USER_PASSWORD` are read on
the hub's **first boot only**; after the database exists they are inert. So if
`.env.beszel-hub` was ever deleted and regenerated while the database survived,
the file now holds a freshly generated password that the database has never
heard of. Reset it from inside the running container:

    podman exec -it beszel-hub /beszel superuser update <email> <new-password>

Or wipe `~/.local/share/beszel/hub` and re-run `install.sh` — `config.yml`
rebuilds the user and the system (verified), at the cost of the metrics history.

**An `Environment=` change in a unit did not take effect.** systemd splits
unquoted `Environment=` values on whitespace. A value containing a space must
be quoted — `Environment="SENSORS=a,GeForce RTX 3050 Ti"` — or it silently
truncates at the first space. This bit this setup once already; verify with
`systemctl --user show beszel-agent -p Environment --value`.

---

## Layout

    beszel/
      quadlets/beszel-hub.container      hub — container, host network, loopback bind
      systemd-user/beszel-agent.service  agent — native, runs as you
      hub/config.yml                     declarative system list (templated)
      scripts/install.sh                 idempotent setup, no sudo
      scripts/beszel.sh                  up | down | status | logs
      scripts/tailscale-serve.sh         publish on :8444
      .env.example                       documents both secret files
      .env.beszel-hub                    600, gitignored — USER_*, AUTO_LOGIN, APP_URL
      .env.beszel-agent                  600, gitignored — KEY

    ~/.local/share/beszel/hub/           SQLite, hub keypair, rendered config.yml
    ~/.local/share/beszel/agent/         agent fingerprint

**Why the hub is a container and the agent is not.** The hub containerises
cleanly and gains from it. The agent does not: it needs host network mode to
read interface stats at all, plus the rootless podman socket and the system
D-Bus socket. A container holding all three isolates nothing — it is the same
process with extra failure modes, the loudest being a Containers tab that comes
up empty because the userns mapping is off, with no error anywhere to explain it.

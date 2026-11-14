# Trove — Running the Application

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.24+ | https://go.dev/dl |
| Bun | 1.3+ | `curl -fsSL https://bun.sh/install \| bash` |
| Podman | 3.4+ | https://podman.io/getting-started/installation |
| podman-compose | latest | `pip install podman-compose` |
| air (hot reload) | latest | `go install github.com/air-verse/air@latest` |

---

## Local Development

Three processes run separately: infrastructure (Postgres + Redis), backend, and UI.

### 1. Environment setup

```bash
cp infra/.env.example infra/.env
# Edit infra/.env if needed (defaults work out of the box)
```

### 2. Start infrastructure

```bash
make dev-infra
```

Starts Postgres 16 and Redis 7 via podman in host-networking mode.

### 3. Apply database migrations

Run once on a fresh database. On an existing database, apply only the files
added since it was last migrated (002 fails if run twice; 003 onwards are safe
to re-run):

```bash
podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/001_initial_schema.sql

podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/002_workout_log_unique.sql

podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/003_sync_schema.sql

# Game lifecycle: status (wishlist/owned/played/dropped), platform, target price
podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/004_game_status.sql

# Price tracking: IGDB metadata, store listings, price history, sale events
podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/005_price_tracking.sql

# Sale alerts: web push subscriptions
podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/006_push.sql

# Device verdicts: which device to play each game on
podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/007_device_verdicts.sql

# One-off: games linked to IGDB automatically take IGDB's title
podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/008_igdb_titles.sql

# Covers and studios filled in by the worker, replaced when you relink a game
podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/009_filled_by_worker.sql

# Weekly picks: games on sale you'd probably like
podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/010_picks.sql
```

### Sale alerts (web push)

The worker sends a notification per game when a wishlisted game goes on
sale. It needs `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY` and `VAPID_SUBJECT` in
`infra/.env`; generate the keys once with `make vapid` (new keys mean every
device has to turn alerts on again).

Push only works from a production build (`src/sw.ts` is the service worker)
on a certificate the browser trusts. To try it against a scratch backend,
serve the build with the tailnet certificate and open
`https://cachyos.tail4f0f0b.ts.net:3100` (tailnet on):

```bash
cd ui && bun run build && \
  TROVE_TLS_CERT=../infra/certs/cachyos.tail4f0f0b.ts.net.crt \
  TROVE_TLS_KEY=../infra/certs/cachyos.tail4f0f0b.ts.net.key \
  TROVE_BACKEND_URL=http://localhost:18080 bunx --bun vite preview --port 3100
```

Browsers that can't receive alerts: CachyOS's `firefox-pure` (push is locked
off) and ungoogled-chromium (no push service). Chrome and Firefox on Android
work.

### Price tracking worker

`backend/cmd/worker` links games to IGDB, fills missing covers and studios,
and checks Steam / eShop prices for every game (only wishlisted games raise
sale alerts). It needs
`IGDB_CLIENT_ID` and `IGDB_CLIENT_SECRET` (a free Twitch developer app) in
`infra/.env`. In production it runs as the `worker` service in
`podman-compose.yml`; locally:

```bash
cd backend && set -a && . ../infra/.env && set +a && \
  UPLOADS_DIR=../uploads go run ./cmd/worker
```

`go run ./cmd/storeprobe "Hades II"` checks the store lookups by hand, and
`go run ./cmd/storeprobe -match "Title|Studio"` shows how a game would be
matched to IGDB.

### Device verdicts

With `ANTHROPIC_API_KEY` in `infra/.env`, the worker also decides which
device (the G15, PS5 or Switch 2) to play each game on, wishlisted games
first and at most 10 a run. It
gathers the platforms (IGDB), Steam's PC requirements and Metacritic score,
ProtonDB, the Steam Deck rating, AreWeAntiCheatYet and web search results,
then asks Claude Sonnet 5.5 (`internal/llm`) for a ranked verdict, about
1–2¢ a game. Verdicts are redone after 90 days or when the game is
relinked; anti-cheat that blocks Linux always rules the PC out.

Web search goes through SearXNG (`SEARXNG_URL`, default
`http://127.0.0.1:8888`, cortex's `searxng.service`). While it's down,
verdicts are made without it and marked incomplete, then redone once it's
reachable (checked every 6 hours). Steam's review endpoint redirects
requests from this machine, so review sentiment comes from the web search.

### Weekly picks

Every Sunday at 09:00 IST (and on the worker's first run after deploying)
the worker chooses five games on sale that you'd probably like, shown in the
**Deals** tab under "Picks for you":

1. Candidates: the 300 best-selling Steam (India) specials and the 120 most
   popular eShop (US) deals, matched to IGDB (Steam by appid, eShop by
   title), leaving out games in your library and recent picks.
2. A taste profile (`internal/picks`) from your linked games: ★5 counts +2
   down to ★1 −2, recent years more; wishlisted and owned +1, dropped −1.5;
   picks you dismiss count against their tags. Candidates are scored on
   shared genres, themes and keywords, IGDB's "similar games" and reviews.
3. Claude Sonnet 5.5 chooses five from the top 15 and writes a pitch for
   each (one call, about 1¢); without it, the top scores are used.

Adding a pick to the wishlist or Owned creates the Trove game, linked to
the picked IGDB game. A run takes about a minute.

### 4. Start the backend

```bash
make dev-backend
```

Runs the Go server on `:8080` with air hot reload. The `UPLOADS_DIR` is set to `../uploads`.

### 5. Start the UI

```bash
make dev-ui
```

Runs the Vite dev server on `:3000` with HMR. API requests (`/api/`) and uploads (`/uploads/`) are proxied to `:8080`.

### Accessing the app

- UI: http://localhost:3000
- UI (HTTPS, if certs exist): https://localhost:3000
- API: http://localhost:8080

### Stop infrastructure

```bash
make dev-down
```

---

## Production

The full stack (Postgres, Redis, backend, nginx + UI) runs in containers.

### 1. Environment setup

```bash
cp infra/.env.example infra/.env
# Edit infra/.env — set strong passwords and correct URLs
```

### 2. Generate TLS certificates (optional, for HTTPS / phone access)

```bash
bash infra/generate-certs.sh
```

Requires [mkcert](https://github.com/FiloSottimo/mkcert). Generates `infra/certs/cert.pem` and `infra/certs/key.pem` covering `trove.local`, `localhost`, `127.0.0.1`, `::1`, and `192.168.137.1`.

To access from a phone, copy the root CA (`mkcert -CAROOT` to find it) to the phone and install it as a trusted certificate.

### 3. Start all services

```bash
make prod-up
```

Builds container images and starts all services. nginx serves:
- HTTP on port **3000**: `/api/` → backend `:8080`, `/uploads/` → backend, `/` → React SPA
- HTTPS on port **3443** (if certs present): same routing, with TLS

### Accessing the app

- UI: http://localhost:3000
- UI (HTTPS): https://localhost:3443

### Stop all services

```bash
make prod-down
```

### Auto-restart and reboot survival

Each service in `infra/podman-compose.yml` has `restart: unless-stopped`. Combined with the host-wide rootless-podman setup (`loginctl enable-linger $USER` and `systemctl --user enable podman-restart.service`, done once for streamcloud), the trove stack:

- Recovers from in-session container crashes
- Comes back automatically after a host reboot

If linger/podman-restart isn't already set up on this machine:

```bash
loginctl show-user $USER | grep Linger          # should show Linger=yes
systemctl --user is-enabled podman-restart      # should be 'enabled'

# If either is missing, run once:
loginctl enable-linger $USER
systemctl --user enable --now podman-restart.service
```

### Verify after a reboot

```bash
podman ps                                       # all four infra_* containers up
curl -kI https://localhost:3443                 # nginx serving (HTTP/2 200)
curl -fsS http://localhost:9000/healthz         # backend reachable (replace with real health endpoint if different)
```

If the stack didn't come up:

```bash
cd ~/Projects/self-hosted-apps/trove && make prod-up
```

---

## Phone Access over Tailscale (any platform)

Tailscale issues publicly-trusted Let's Encrypt certs for your `<host>.<tailnet>.ts.net` MagicDNS hostname. This is the cleanest way to access trove from a phone over the internet — no CA install on the phone, and Chrome will offer **Install app** for the PWA.

### 1. Enable HTTPS in the admin console

Tailscale admin console → **DNS** → enable **MagicDNS** and **HTTPS Certificates**.

### 2. Issue the cert

```bash
sudo tailscale cert --cert-file infra/certs/<host>.<tailnet>.ts.net.crt \
                    --key-file  infra/certs/<host>.<tailnet>.ts.net.key \
                    <host>.<tailnet>.ts.net
sudo chown $USER:$USER infra/certs/<host>.<tailnet>.ts.net.{crt,key}
```

### 3. Point nginx at the cert

Edit `infra/nginx.conf` `ssl_certificate` / `ssl_certificate_key` to:
`/etc/nginx/certs/<host>.<tailnet>.ts.net.{crt,key}` (the volume mount inside the container), then `make prod-down && make prod-up`.

### 4. Auto-renewal (90-day expiry)

Install the staged systemd timer once:

```bash
sudo install -m 755 infra/systemd/trove-tailscale-cert-renew /usr/local/bin/
sudo install -m 644 infra/systemd/trove-tailscale-cert.service /etc/systemd/system/
sudo install -m 644 infra/systemd/trove-tailscale-cert.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now trove-tailscale-cert.timer
```

The timer runs weekly. `tailscale cert` no-ops until ~14 days before expiry, then re-issues and reloads nginx in `infra_ui_1`. Update `HOSTNAME` in the script if your tailnet/host changes.

### 5. MagicDNS troubleshooting (Linux + NetworkManager)

If `tailscale status` warns "MagicDNS will probably not work", make systemd-resolved fully own DNS:

```bash
sudo tee /etc/NetworkManager/conf.d/dns.conf <<'EOF'
[main]
dns=systemd-resolved
systemd-resolved=false
EOF

sudo ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
sudo systemctl restart systemd-resolved NetworkManager tailscaled
sudo tailscale down && sudo tailscale up --accept-dns=true

resolvectl status tailscale0                    # should list 100.100.100.100 and ~<tailnet>.ts.net
resolvectl query <host>.<tailnet>.ts.net        # should resolve
```

---

## Phone Access over Wi-Fi Hotspot (WSL2)

WSL2 runs in a virtual network that isn't directly reachable from the Windows host network. To access Trove from a phone connected to your laptop's hotspot:

### 1. Generate and install TLS certificates

```bash
bash infra/generate-certs.sh
```

Copy the root CA to your phone and install it as a trusted certificate (Settings → Security → Install certificate on Android).

### 2. Set up WSL2 → Windows port forwarding

The WSL2 IP changes on every reboot, so this must be re-run each time. Run in **PowerShell as Administrator**:

```powershell
.\infra\wsl-portforward.ps1
```

Or from WSL:

```bash
make port-forward
```

This forwards ports 3000, 3443, and 8080 from all Windows interfaces (including the hotspot at `192.168.137.1`) to the WSL2 VM, and adds a Windows Firewall allow rule.

### 3. Enable Mobile Hotspot

Windows Settings → Network & Internet → Mobile hotspot → On

### 4. Start services

```bash
# Dev mode
make dev-infra && make dev-backend && make dev-ui

# Or production mode
make prod-up
```

### 5. Access from phone

| Mode | URL |
|------|-----|
| Dev (HTTP) | `http://192.168.137.1:3000` |
| Prod (HTTP) | `http://192.168.137.1:3000` |
| Prod (HTTPS) | `https://192.168.137.1:3443` |

### Post-reboot checklist

1. Open **PowerShell as Administrator** → run `.\infra\wsl-portforward.ps1`
2. Enable Windows Mobile Hotspot
3. Start services (`make dev-infra`, `make dev-backend`, `make dev-ui` — or `make prod-up`)

---

## Phone Access over Wi-Fi Hotspot (Arch Linux / CachyOS)

On native Linux, services bind directly to all interfaces — no port forwarding script is needed.

### 1. Generate TLS certificates (if not already done)

```bash
bash infra/generate-certs.sh
```

Certs cover `trove.local`, `localhost`, `127.0.0.1`, `::1`, `192.168.137.1`, and `10.42.0.1`.

Copy the root CA to your phone and install it as a trusted certificate. The easiest way is to serve it temporarily over HTTP while the hotspot is active:

```bash
cd "$(mkcert -CAROOT)" && python3 -m http.server 8888
```

Then open `http://10.42.0.1:8888/rootCA.pem` in the phone browser to download it, then stop the server (`Ctrl+C`).

Android: Settings → Security → Encryption & credentials → Install a certificate → CA certificate → select the downloaded file.

### 2. Start the hotspot

```bash
nmcli dev wifi hotspot ifname wlan0 ssid Trove password "<your-password>"
```

Verify the gateway IP (usually `10.42.0.1`):

```bash
ip -4 addr show wlan0
```

> **Note:** Starting the hotspot switches the physical WiFi card to AP mode, disconnecting the laptop from any existing WiFi network. This is a single-card hardware limitation on Linux (unlike Windows which uses a virtual adapter). Only enable the hotspot when you don't need the laptop's WiFi connection.

### 3. Open firewall ports

If `firewalld` is active:

```bash
sudo firewall-cmd --permanent --add-port=3000/tcp
sudo firewall-cmd --permanent --add-port=3443/tcp
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

If `ufw` is active instead, two rules are required — the port rules alone are not enough, phones also need DHCP (to get an IP) which requires allowing all traffic on the hotspot interface:

```bash
sudo ufw allow 3000/tcp && sudo ufw allow 3443/tcp && sudo ufw allow 8080/tcp
sudo ufw allow in on wlan0
sudo ufw reload
```

### 4. Start services

```bash
# Dev mode
make dev-infra && make dev-backend && make dev-ui

# Or production mode
make prod-up
```

### 5. Access from phone

| Mode | URL |
|------|-----|
| Dev (HTTP) | `http://10.42.0.1:3000` |
| Prod (HTTP) | `http://10.42.0.1:3000` |
| Prod (HTTPS) | `https://10.42.0.1:3443` |

---

## Notes

- All services use `network_mode: host` (required for podman on WSL2 — CNI DNS doesn't work)
- Upload files are persisted in `uploads/` (gitignored, mounted as a volume)
- Postgres data is persisted in a named podman volume (`postgres_data`)
- TLS is opt-in: leave `TLS_CERT_FILE` and `TLS_KEY_FILE` empty in `.env` to disable (HTTP only)
- Vite dev server auto-detects certs in `infra/certs/` and enables HTTPS if found
- The Makefile `dev-backend` target sources `infra/.env` and overrides TLS paths for local dev

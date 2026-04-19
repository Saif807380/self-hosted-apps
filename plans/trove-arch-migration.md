# Trove Migration: WSL2 → CachyOS (Arch Linux)

## Context
You've set up CachyOS on an external SSD and want it to be your primary machine. The Trove app currently runs under WSL2 Ubuntu and needs to move to CachyOS with zero data loss. Three concrete asks:

1. Install the runtime stack on Arch (Go, Bun, Podman).
2. Move the Postgres database and the uploaded images so the app comes up on CachyOS with identical data.
3. Reconfigure networking (certs, hotspot, firewall) so the mobile PWA still works when you enable the CachyOS hotspot.

The plan is split into two **sequential** parts because you can't run Windows and CachyOS at the same time:

- **Part A — on Windows / WSL2** (do all of this first, while booted into Windows). Back up data, make repo edits that Arch will need, stage everything to a USB drive.
- **Part B — on CachyOS** (reboot, boot from the external SSD). Install the stack, clone the repo, restore data from USB, regenerate certs for the Arch hotspot, verify.

The WSL2 install is left fully intact — Part A is non-destructive, so you can fall back to WSL2 at any time.

The Trove code itself is already OS-agnostic — relative `/api` URLs on the frontend, pure-Go backend with `CGO_ENABLED=0`, and `network_mode: host` containers. The migration is about environment, data, and hotspot networking — not app code.

---

## Current WSL2 Baseline (what we're migrating from)

- **Postgres data**: rootless podman named volume `infra_postgres_data` at `/home/saifkazi/.local/share/containers/storage/volumes/infra_postgres_data/_data`
- **Uploads**: bind mount at `trove/uploads/` — 23 MB, 83 image files (UUID-named .jpg/.png/.webp)
- **Redis**: in-memory, ephemeral (nothing to migrate)
- **Certs**: mkcert-generated at `trove/infra/certs/{cert,key}.pem`, SANs include `192.168.137.1` (Windows Mobile Hotspot gateway — WSL2-specific)
- **Port forwarding**: `trove/infra/wsl-portforward.ps1` (PowerShell) plugs ports 3000/3443/8080 from Windows into the WSL2 VM on every reboot — not needed on native Linux
- **Stack versions**: Go 1.25.0 (go.mod), Bun ≥1.1 (container is `bun:1.1-alpine`), Postgres 16-alpine, Redis 7-alpine

---

# PART A — Windows / WSL2

*All steps in this section run on your current Windows machine inside WSL2, with Trove's stack up. Finish Part A completely, eject the USB, shut down, then reboot into CachyOS for Part B.*

## A1 — Back up the Postgres database

```bash
cd ~/development/self-hosted-apps/trove
PG=$(podman ps --format '{{.Names}}' | grep postgres)
podman exec "$PG" pg_dump -U lms -d lms --clean --if-exists > /tmp/trove-db.sql
```
`--clean` makes the dump idempotent: it drops tables before recreating, so it safely restores into a freshly-initialised Postgres on Arch.

## A2 — Archive the uploads directory

```bash
tar -czf /tmp/trove-uploads.tar.gz -C ~/development/self-hosted-apps/trove uploads
```
This captures all 83 image files (~23 MB).

## A3 — Back up the current `.env`

```bash
cp ~/development/self-hosted-apps/trove/infra/.env /tmp/trove-env.backup
```
You'll reuse this as a starting point on CachyOS and rotate the DB password there.

## A4 — Capture a row-count baseline

```bash
podman exec "$PG" psql -U lms -d lms -c "\
  SELECT 'books' tbl, COUNT(*) FROM books UNION ALL \
  SELECT 'video_games', COUNT(*) FROM video_games UNION ALL \
  SELECT 'travel_locations', COUNT(*) FROM travel_locations UNION ALL \
  SELECT 'workout_logs', COUNT(*) FROM workout_logs;" > /tmp/trove-rowcounts.txt
```
You'll diff this against CachyOS after restore to confirm zero-loss migration.

## A5 — Make the repo edits Arch will need, commit and push

Doing these on WSL2 and pushing to GitHub means the Arch clone in Part B already has them. Two files change; both are additive (WSL2 keeps working).

**Edit 1 — `trove/infra/generate-certs.sh`**: add `10.42.0.1` (NetworkManager's default hotspot gateway on Arch) to the SAN list alongside the existing WSL2 IP. Resulting `mkcert` invocation:
```bash
mkcert \
  -cert-file "$CERT_DIR/cert.pem" \
  -key-file "$CERT_DIR/key.pem" \
  trove.local localhost 127.0.0.1 ::1 192.168.137.1 10.42.0.1
```

**Edit 2 — `trove/DEVELOPMENT.md`**: add a new subsection under "Phone Access over Wi-Fi Hotspot" titled **"Arch Linux (CachyOS) — NetworkManager hotspot"**, covering:
- `nmcli dev wifi hotspot ifname <iface> ssid <name> password <pw>` to start the hotspot
- Gateway IP `10.42.0.1` and phone URL `https://10.42.0.1:3443`
- A note that no port-forward script is needed on native Linux
- Firewall setup (`firewall-cmd` / `ufw` commands — see Step B7)

Leave the existing WSL2 subsection in place — dual-maintain means Windows stays documented.

Then commit and push:
```bash
cd ~/development/self-hosted-apps
git checkout -b chore/arch-migration-support
git add trove/infra/generate-certs.sh trove/DEVELOPMENT.md
git commit -m "chore(trove): add Arch hotspot IP and NetworkManager docs"
git push -u origin chore/arch-migration-support
# Merge to main via GitHub PR so a fresh clone on CachyOS pulls it
```
(If you prefer not to push before testing on Arch, skip the push — you can copy the modified files to USB alongside the backups and apply them manually on CachyOS.)

## A6 — Copy everything to the USB drive

With the USB stick mounted (typically Windows auto-mounts it under `/mnt/d/`, `/mnt/e/`, etc.):
```bash
USB=/mnt/d   # adjust
cp /tmp/trove-db.sql /tmp/trove-uploads.tar.gz /tmp/trove-env.backup /tmp/trove-rowcounts.txt "$USB/"

# Optional: also copy mkcert's CA so CachyOS can reuse the same CA (phone keeps trusting it)
cp -r "$(mkcert -CAROOT)" "$USB/mkcert-CA-wsl"

sync
```
Eject safely, shut down Windows, boot into CachyOS.

---

# PART B — CachyOS (Arch Linux)

*Everything from here on runs on the CachyOS install. WSL2 is powered off.*

## B1 — Install the runtime stack

Runtime packages (official repos):
```bash
sudo pacman -S --needed podman podman-compose go mkcert nss networkmanager postgresql-libs
```
- `podman` + `podman-compose` — container runtime and compose shim
- `go` — Go 1.25+ on current Arch; needed for backend and for installing `air`
- `mkcert` + `nss` — TLS cert generation and browser trust store integration
- `networkmanager` — likely already installed; provides `nmcli` hotspot
- `postgresql-libs` — optional, gives you `psql` / `pg_dump` on the host for debugging

Bun (official installer — not in Arch repos as a stable package):
```bash
curl -fsSL https://bun.sh/install | bash
# Add ~/.bun/bin to PATH in ~/.bashrc or ~/.zshrc, then re-source it
```

Dev-only tools (after Go is on PATH):
```bash
go install github.com/air-verse/air@latest
# air lands in ~/go/bin — make sure that's on PATH too
```

Rootless podman sanity check:
```bash
podman info | grep -i rootless      # expect: rootless: true
cat /etc/subuid /etc/subgid           # confirm your user has ranges
podman run --rm docker.io/library/hello-world
```
If rootless is broken: `sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 $USER`, then log out/in.

## B2 — Clone the repo

```bash
mkdir -p ~/development
git clone https://github.com/Saif807380/self-hosted-apps.git ~/development/self-hosted-apps
cd ~/development/self-hosted-apps/trove
```
If you pushed the A5 branch, merge it to main first (or check out `chore/arch-migration-support`). `uploads/`, `infra/certs/`, and `infra/.env` are gitignored — those come from the USB next.

## B3 — Pull backups off the USB drive

Mount (usually auto-mounted by GNOME/KDE under `/run/media/saifkazi/<label>`):
```bash
USB=/run/media/saifkazi/<label>      # check lsblk or the file manager
cp "$USB"/trove-{db.sql,uploads.tar.gz,env.backup,rowcounts.txt} ~/

# Optional: copy the WSL2 mkcert CA so both hosts issue certs under the same CA
# (phone keeps trusting one rootCA.pem instead of needing two)
CACHY_CA_DIR="$(mkcert -CAROOT)"
mkdir -p "$CACHY_CA_DIR"
cp "$USB"/mkcert-CA-wsl/* "$CACHY_CA_DIR/"
```

## B4 — Restore data on CachyOS (with DB password rotation)

The old `lms:lms` credential stays on the network once you start using the hotspot, so we're rotating it as part of the migration.

```bash
cd ~/development/self-hosted-apps/trove

# B4a) Start from the backed-up .env
cp ~/trove-env.backup infra/.env

# Generate a strong password
NEW_PW=$(openssl rand -base64 24)
echo "$NEW_PW"    # save this to your password manager

# Edit infra/.env — set both lines to the new value:
#   POSTGRES_PASSWORD=<NEW_PW>
#   DATABASE_URL=postgres://lms:<NEW_PW>@127.0.0.1:5432/lms?sslmode=disable
# (Use an editor — be careful with $ / # / & if the generated string contains them)

# B4b) Spin up Postgres + Redis (fresh empty volume on Arch)
make dev-infra
sleep 5   # let Postgres finish initdb + apply init-db.sql
#          initdb creates the 'lms' role with the new password because
#          podman-compose reads the updated .env at container start.

# B4c) Restore the dump. pg_dump --clean drops and recreates tables;
#      it does NOT touch roles, so the new password stays in place.
PG=$(podman ps --format '{{.Names}}' | grep postgres)
podman exec -i -e PGPASSWORD="$NEW_PW" "$PG" psql -U lms -d lms < ~/trove-db.sql

# B4d) Restore uploads
tar -xzf ~/trove-uploads.tar.gz -C ~/development/self-hosted-apps/trove/

# B4e) Verify against the WSL2 baseline
ls uploads/ | wc -l                 # should match WSL2 count (~83)
podman exec -e PGPASSWORD="$NEW_PW" "$PG" psql -U lms -d lms -c "\
  SELECT 'books' tbl, COUNT(*) FROM books UNION ALL \
  SELECT 'video_games', COUNT(*) FROM video_games UNION ALL \
  SELECT 'travel_locations', COUNT(*) FROM travel_locations UNION ALL \
  SELECT 'workout_logs', COUNT(*) FROM workout_logs;"
# Compare the output against ~/trove-rowcounts.txt — numbers should match exactly.
```

`infra/.env` is gitignored, so the rotated password stays local to the CachyOS checkout. The WSL2 side keeps its own `.env` untouched when you boot back into Windows.

## B5 — Generate TLS certs

The cert now covers both hotspot IPs (thanks to the A5 edit): `trove.local localhost 127.0.0.1 ::1 192.168.137.1 10.42.0.1`.

Before running `mkcert -install`: decide whether to reuse the WSL2 CA (copied via B3, phone already trusts it) or create a fresh CachyOS CA (phone will need a new rootCA.pem installed).

```bash
mkcert -install                  # installs the CA into the system trust store
bash infra/generate-certs.sh     # writes infra/certs/{cert,key}.pem
```

If you created a fresh CA (didn't copy from USB in B3), transfer the new rootCA.pem to your phone:
```bash
mkcert -CAROOT    # prints the directory containing rootCA.pem
# Transfer via KDE Connect, USB, email to yourself, etc.
# Android: Settings → Security → Install a certificate → CA certificate
```

## B6 — Confirm the Arch hotspot IP

NetworkManager's built-in hotspot defaults to `10.42.0.1`. Confirm the first time:
```bash
nmcli dev wifi hotspot ifname wlan0 ssid Trove password "<strong-pw>"
ip -4 addr show wlan0           # note the IPv4 (usually 10.42.0.1/24)
```
If your gateway differs, update the SAN in `generate-certs.sh` and regenerate (`bash infra/generate-certs.sh`).

## B7 — Firewall configuration

CachyOS is typically **firewalld**-active by default. Check:
```bash
sudo systemctl is-active firewalld    # "active" or "inactive"
sudo systemctl is-active ufw
```

Open the three Trove ports (only needed if a firewall is active):
```bash
# firewalld
sudo firewall-cmd --permanent --add-port=3000/tcp
sudo firewall-cmd --permanent --add-port=3443/tcp
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload

# or ufw
sudo ufw allow 3000/tcp && sudo ufw allow 3443/tcp && sudo ufw allow 8080/tcp
```

No port forwarding is needed — on native Linux with `network_mode: host`, the containers bind directly to all interfaces (including the hotspot one).

## B8 — Bring the stack up

```bash
cd ~/development/self-hosted-apps/trove
make prod-up        # full containerized stack
# or, for a dev loop:
# make dev-infra
# make dev-backend     (in a second terminal)
# make dev-ui          (in a third terminal)
```

## B9 — End-to-end verification

**On CachyOS:**
- `podman ps` → 4 containers (postgres, redis, backend, ui) or 2 (dev-infra)
- `curl -k https://localhost:3443/api/health` → expect 200 OK
- Open `https://localhost:3443` in a browser → UI loads without cert warning, all books/games/travel/workouts present, images render (exercises the uploads/ bind mount)

**On phone (over hotspot):**
- Start hotspot: `nmcli dev wifi hotspot ifname wlan0 ssid Trove password "<pw>"`
- Connect phone → verify phone shows `10.42.0.x` IP in its WiFi details
- Open `https://10.42.0.1:3443` in mobile browser → no cert warning (rootCA installed), PWA loads
- Add the app to home screen (PWA install)
- Create a new book on the phone → within ~10 s (sync-scheduler interval) it appears on the laptop
- Attach an image on the phone → verify it propagates to laptop on next sync

If the cert fails on phone: re-check that the rootCA you installed matches the CA you ran `mkcert -install` with (either the copied WSL2 one or a fresh CachyOS one — not both mixed).

---

## Critical files touched

| Path | Where edited | Change |
|------|--------------|--------|
| `trove/infra/generate-certs.sh` | WSL2 (A5) | Add `10.42.0.1` to SAN list alongside `192.168.137.1` |
| `trove/DEVELOPMENT.md` | WSL2 (A5) | Add Arch/NetworkManager subsection; leave WSL2 section intact |
| `trove/infra/.env` *(gitignored)* | CachyOS (B4a) | New strong `POSTGRES_PASSWORD`, matching `DATABASE_URL` |

No deletions. `wsl-portforward.ps1`, the `port-forward` Makefile target, and the WSL2 docs all stay. Backend, UI, compose, nginx.conf, Containerfiles, and init-db.sql are untouched.

---

## Reference — existing commands you'll re-use

- `make dev-infra` / `make dev-down` — postgres + redis up/down (`trove/Makefile:9,13`)
- `make dev-backend` — `air` hot-reload Go server, sources `infra/.env` (`trove/Makefile:17`)
- `make dev-ui` — Vite HMR on `:3000`, proxies `/api` + `/uploads` to `:8080` (`trove/Makefile:26`)
- `make prod-up` / `make prod-down` — full stack containerized (`trove/Makefile:30,33`)
- `bash infra/generate-certs.sh` — mkcert wrapper producing `cert.pem` + `key.pem` (`trove/infra/generate-certs.sh`)

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

Run once on a fresh database (or after adding new migration files):

```bash
podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/001_initial_schema.sql

podman exec -i $(podman ps --format '{{.Names}}' | grep postgres) \
  psql -U lms -d lms -f - < backend/migrations/002_workout_log_unique.sql
```

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

## Notes

- All services use `network_mode: host` (required for podman on WSL2 — CNI DNS doesn't work)
- Upload files are persisted in `uploads/` (gitignored, mounted as a volume)
- Postgres data is persisted in a named podman volume (`postgres_data`)
- TLS is opt-in: leave `TLS_CERT_FILE` and `TLS_KEY_FILE` empty in `.env` to disable (HTTP only)
- Vite dev server auto-detects certs in `infra/certs/` and enables HTTPS if found
- The Makefile `dev-backend` target sources `infra/.env` and overrides TLS paths for local dev

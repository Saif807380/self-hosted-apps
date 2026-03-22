.PHONY: dev-infra dev-backend dev-ui dev-down prod-up prod-down port-forward

# Set up WSL2 → Windows port forwarding (requires PowerShell as Admin)
port-forward:
	powershell.exe -ExecutionPolicy Bypass -File infra/wsl-portforward.ps1

# Start only postgres + redis for local development
dev-infra:
	cd infra && podman-compose -f podman-compose.dev.yml up -d

# Stop dev infrastructure
dev-down:
	cd infra && podman-compose -f podman-compose.dev.yml down

# Run backend with hot reload (install air first: go install github.com/air-verse/air@latest)
# Sources infra/.env, overrides container paths with local dev paths
dev-backend:
	cd backend && set -a && . ../infra/.env && set +a && \
		UPLOADS_DIR=../uploads \
		TLS_CERT_FILE=../infra/certs/cert.pem \
		TLS_KEY_FILE=../infra/certs/key.pem \
		air

# Run UI dev server with Vite HMR (proxies /api and /uploads to localhost:8080)
dev-ui:
	cd ui && bun run dev

# Production: full stack in containers
prod-up:
	cd infra && podman-compose up -d --build

prod-down:
	cd infra && podman-compose down

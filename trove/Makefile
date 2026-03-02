.PHONY: dev-infra dev-backend dev-ui dev-down prod-up prod-down

# Start only postgres + redis for local development
dev-infra:
	cd infra && podman-compose -f podman-compose.dev.yml up -d

# Stop dev infrastructure
dev-down:
	cd infra && podman-compose -f podman-compose.dev.yml down

# Run backend with hot reload (install air first: go install github.com/air-verse/air@latest)
dev-backend:
	cd backend && UPLOADS_DIR=../uploads air

# Run UI dev server with Vite HMR (proxies /api and /uploads to localhost:8080)
dev-ui:
	cd ui && bun run dev

# Production: full stack in containers
prod-up:
	cd infra && podman-compose up -d --build

prod-down:
	cd infra && podman-compose down

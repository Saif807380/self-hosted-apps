# Self-Hosted Apps — Project Design Document

## Objective

A monorepo housing multiple independently deployable, fully self-hosted applications. Each application runs entirely on local infrastructure with no dependency on external managed services.

---

## Repository Structure

```
self-hosted-apps/
├── commons/                  # Shared libraries by language
│   ├── golang-commons/
│   ├── python-commons/
│   └── ...
├── <app-name>/               # One top-level directory per application
│   ├── ui/                   # Frontend (if applicable)
│   ├── backend/              # Backend service(s)
│   ├── infra/                # App-specific infra config (compose files, etc.)
│   └── plans/                # Architecture and implementation plan for the app
├── scripts/                  # Commands to bring up one or all applications
└── plans/                    # Project-level design documents
```

---

## Core Principles

### Self-Hosted
Every infrastructure component (databases, caches, message queues, etc.) runs locally. No external managed services. If multiple applications share a common component type (e.g., Postgres), each app gets its own logical database on a shared local instance.

### Containerized
Each application and its dependencies are fully containerized using Podman. A single command per app (or for all apps) brings up the complete stack — UI, backend services, and all infra.

### Co-located Configuration
All dependencies, tests, config files, and infra definitions live within the application's own directory. There is no shared global config.

### Shared Commons
Common code that is reusable across applications is extracted into the `commons/` directory, organized by language/runtime.

---

## Applications

Each application has its own detailed plan document covering functional requirements, architecture, and implementation. The list below reflects applications planned so far.

| # | Application | Description |
|---|---|---|
| 1 | Life Library Management System | Tracks personal media consumption — books, games, places visited, etc. Presents a browsable UI organized by category. |
| 2 | Photography Portfolio Website | Digital portfolio for managing and displaying photo collections. |

---

## Tech Stack

| Layer | Technologies |
|---|---|
| Frontend | React, Bun.js |
| Backend | Golang |
| Communication | gRPC |
| Databases | SQL and/or NoSQL (per application need) |
| Caching | Redis |
| Containerization | Podman |

Specific technology choices per application are documented in each app's plan.

---

## Deployment

- Each application has an `infra/` directory containing its container definitions and orchestration config.
- A top-level `scripts/` directory provides commands to start a single application or all applications.
- Running the start command for an app brings up all required services in the correct order so the application is immediately usable.

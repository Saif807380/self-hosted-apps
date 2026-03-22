# Trove Mobile Sync — Detailed Implementation Plan

## Context

Trove is a personal library/journaling app running locally via podman containers (Go backend, React+Bun UI, PostgreSQL, Redis). The goal is to access it from a phone **without deploying to the internet**. The laptop creates a Wi-Fi hotspot, the phone connects to it, and data syncs via an offline-first architecture. Both laptop and phone run the same PWA with IndexedDB as the local store, syncing to the server when connected.

### Current State Summary
- **IDs**: Already UUID everywhere — no migration needed
- **Soft deletes**: Not implemented — all deletes are hard `DELETE FROM`
- **`updated_at`**: Exists on `books`, `video_games`, `travel_locations` — missing on 6 other entity tables
- **Server**: HTTP only, port 8080, no TLS
- **Frontend**: Direct API calls via hooks, no offline capability, no PWA
- **Infra**: Host networking on WSL2, nginx on port 3000 proxying to backend

---

## Phase 1: Database Schema Evolution

**Goal**: Add `updated_at` and `deleted` columns to all entity tables. Existing app continues to work unchanged.

### 1.1 Create migration `backend/migrations/003_sync_schema.sql`
- Add `updated_at TIMESTAMPTZ DEFAULT now()` to: `tags`, `collections`, `tourist_spots`, `workout_types`, `exercises`, `workout_logs`
- Add `deleted BOOLEAN NOT NULL DEFAULT false` to all 9 entity tables: `books`, `video_games`, `travel_locations`, `tags`, `collections`, `tourist_spots`, `workout_types`, `exercises`, `workout_logs`
- Add `updated_at` indexes on all 9 tables for efficient delta queries
- Create `sync_metadata` table (`key VARCHAR(100) PK`, `value TEXT`)
- Junction tables (`book_years_read`, `book_tags`, `collection_books`, `game_years_played`) do NOT get these columns — they follow parent entity sync

### 1.2 Update `backend/internal/model/models.go`
- Add `Deleted bool` to all entity structs
- Add `UpdatedAt time.Time` to `Tag`, `Collection`, `TouristSpot`, `WorkoutType`, `Exercise`, `WorkoutLog` (the others already have it)

### 1.3 Update all DAO files
- **`dao/books.go`**: Append `AND deleted = false` to all SELECTs. Change `Delete` from `DELETE FROM` to `UPDATE SET deleted = true, updated_at = now()`. Same for tags, collections.
- **`dao/games.go`**: Same pattern for video games
- **`dao/travel.go`**: Same for locations and tourist spots
- **`dao/workouts.go`**: Same for workout types, exercises, workout logs

### 1.4 Update `infra/init-db.sql`
- Add the new columns so fresh deployments get the schema right

### Verification
- Apply migration to dev DB
- Run `make dev-backend` + `make dev-ui`, verify all CRUD still works
- Delete a book → confirm row has `deleted = true` (not removed)

### Gotchas
- Junction table `replaceBookYears`/`replaceBookTags` still use hard DELETE — this is correct, junction rows follow parent
- Redis cache invalidation already fires on delete path — no change needed

---

## Phase 2: Backend Sync API

**Goal**: Add a `SyncService` via ConnectRPC for batch sync (push/pull changes), plus REST endpoints for image sync only (multipart not supported by ConnectRPC). This keeps the sync API consistent with the existing 33 RPCs.

### 2.1 New proto: `backend/proto/trove/v1/sync/sync.proto`
- Define `SyncService` with two RPCs:
  - `PushChanges(PushChangesRequest) returns (PushChangesResponse)` — client sends mutations
  - `PullChanges(PullChangesRequest) returns (PullChangesResponse)` — client pulls delta since timestamp
- `PushChangesRequest` contains repeated fields for each entity type (books, tags, collections, video_games, etc.) + junction table entries (book_years_read, book_tags, collection_books, game_years_played)
- `PushChangesResponse` returns `server_time` (ISO8601 string) — client stores as `lastSync` cursor
- `PullChangesRequest` has `since` (string, ISO8601 timestamp)
- `PullChangesResponse` returns the same structure as `PushChangesRequest` (all entity arrays) + `server_time`
- Reuse existing proto messages where possible (e.g., `Book`, `VideoGame` from existing protos) — add `deleted` field to each
- Configure `connect.WithMaxBytes(32 * 1024 * 1024)` (32MB) on the sync handlers to accommodate large full-sync payloads

### 2.2 Generate code
- Run `buf generate` to produce Go server stubs + TypeScript client in `backend/gen/` and `ui/src/services/`
- The generated `SyncServiceHandler` follows the same pattern as existing `BookServiceHandler`, etc.

### 2.3 New file: `backend/internal/dao/sync.go`
- `SyncStore` with `GetChangesSince(ctx, since time.Time) (ChangeSet, error)` — queries all 9 entity tables + 4 junction tables for `WHERE updated_at > $1` (entity tables) or by joining to parent (junction tables). **Includes soft-deleted records** so clients learn about deletions.
- `ApplyChanges(ctx, changes ChangeSet) error` — single transaction, `SET CONSTRAINTS ALL DEFERRED`, upserts in topological order:
  1. `tags`, `collections`, `workout_types` (no FK deps)
  2. `books`, `video_games`, `travel_locations`
  3. `exercises` (→ workout_types), `tourist_spots` (→ travel_locations)
  4. `workout_logs` (→ exercises)
  5. Junction tables: delete-all-for-parent + re-insert
- UPSERT pattern: `INSERT ... ON CONFLICT (id) DO UPDATE ... WHERE EXCLUDED.updated_at > table.updated_at`
- Special case: `workout_logs` has unique on `(exercise_id, week_number)` — use `ON CONFLICT (id)` with deferred constraints

### 2.4 New file: `backend/internal/controller/sync.go`
- ConnectRPC handler implementing `SyncServiceHandler` interface (generated from proto)
- `PushChanges` — maps proto request to DAO `ChangeSet`, calls `ApplyChanges`, returns `server_time`
- `PullChanges` — calls `GetChangesSince`, maps DAO `ChangeSet` to proto response + `server_time`
- **Image endpoints remain REST** (registered as plain HTTP handlers on the mux):
  - `POST /sync/images` — multipart upload with `id` and `checksum` fields, atomic write (`.tmp` → rename after SHA-256 verify)
  - `GET /sync/images/manifest` — returns `[{id, path, checksum, size}]` for all files in uploads dir

### 2.5 New files: `backend/internal/service/sync.go`, update `service/services.go`
- `SyncService` wrapping `SyncStore`, invalidates Redis caches after `ApplyChanges`

### 2.6 Modify `backend/internal/controller/router.go`
- Register `SyncServiceHandler` via ConnectRPC (same pattern as existing services)
- Register image REST routes on the mux

### 2.7 Modify `backend/internal/dao/stores.go`
- Add `Sync *SyncStore` field

### 2.8 Refactor `backend/internal/controller/upload.go`
- Extract atomic-write + checksum logic into shared helper for both existing upload and sync image upload

### Verification
- Use the generated TypeScript client (or `buf curl`) to call `PullChanges` with `since=2000-01-01T00:00:00Z` — returns all data
- Call `PushChanges` with a new book (client-generated UUID) → appears in UI
- Create book via UI → appears in `PullChanges` response
- `curl POST /sync/images` with multipart file + checksum → file written atomically

### Gotchas
- `updated_at` LWW relies on clock accuracy — fine for single-user single-server
- Junction table sync: client must send full set of junction rows per parent (partial = data loss, same as existing pattern)
- Max message size: default ConnectRPC limit is 4MB. Set `connect.WithMaxBytes(32<<20)` on sync handlers for full-sync payloads
- The generated TypeScript client for `SyncService` will use the same `rpc()` helper in `src/services/client.ts` — the sync engine calls it instead of hand-crafting fetch requests

---

## Phase 3: TLS & Network Infrastructure

**Goal**: Enable HTTPS so the phone's browser accepts the connection over the hotspot.

### 3.1 New: `infra/generate-certs.sh`
```bash
mkcert -install
mkcert -cert-file infra/certs/cert.pem -key-file infra/certs/key.pem \
  trove.local localhost 127.0.0.1 ::1 192.168.137.1
```

### 3.2 Modify `backend/internal/config/config.go`
- Add `TLSCertFile`, `TLSKeyFile` fields (env vars `TLS_CERT_FILE`, `TLS_KEY_FILE`)
- Add `TLSEnabled() bool` method — returns true only when both env vars are non-empty

### 3.3 Modify `backend/cmd/server/main.go`
- Conditional: if TLS env vars set → `ListenAndServeTLS`, else → `ListenAndServe`
- When TLS enabled, skip h2c wrapper (native HTTP/2 via ALPN)

### 3.4 Modify `infra/nginx.conf`
- Add HTTPS server block on port 3443 with cert/key
- Keep HTTP on 3000 for backward compatibility

### 3.5 Modify `infra/podman-compose.yml`
- Mount `certs/` volume into backend and nginx containers
- Add TLS env vars

### 3.6 New: `infra/certs/.gitignore` (ignore all certs)

### 3.7 Update `infra/.env.example` with TLS vars

### 3.8 New: `infra/wsl-portforward.ps1`
- PowerShell script (run as Administrator) that:
  - Auto-detects WSL2 IP via `wsl hostname -I`
  - Sets up `netsh interface portproxy` forwarding for ports 3000, 3443, 8080 from `0.0.0.0` to the WSL2 IP
  - Cleans up legacy firewall rules and adds a single "Trove WSL2 Ports" allow rule
- Must be re-run after every reboot (WSL2 IP changes)

### 3.9 Modify `Makefile`
- `dev-backend`: sources `infra/.env`, overrides `TLS_CERT_FILE` and `TLS_KEY_FILE` with local paths (`../infra/certs/`) for dev (`.env` has container paths for prod)
- New `port-forward` target: runs `wsl-portforward.ps1` via `powershell.exe`

### 3.10 Modify `ui/vite.config.ts`
- Auto-detect certs in `../infra/certs/` — if present, enable HTTPS on Vite dev server
- Switch proxy target to `https://localhost:8080` when certs present, with `secure: false` for self-signed
- Bind to `host: '0.0.0.0'` so the dev server is reachable from external IPs (hotspot)

### 3.11 Update `DEVELOPMENT.md`
- Full post-reboot checklist for phone access
- Document port forwarding setup, TLS cert generation, and hotspot access URLs

### Verification
- `https://localhost:8080` serves backend (when certs exist)
- `https://localhost:3000` serves Vite dev UI (when certs exist)
- Phone on hotspot → `http://192.168.137.1:3000` loads the app (dev mode)
- Phone on hotspot → `https://192.168.137.1:3443` loads the app (prod mode, after rootCA installed)

### Gotchas
- WSL2 networking: hotspot IPs are on the Windows host, WSL2 has its own virtual network — port forwarding required
- Skip mDNS for now — use raw IP, mDNS on WSL2 is unreliable
- Install rootCA on phone: transfer `rootCA.pem`, install in OS trusted cert store
- **Dev vs prod TLS paths**: `.env` must use container paths (`/certs/cert.pem`) for prod containers. Makefile overrides with relative paths (`../infra/certs/cert.pem`) for local dev. Do NOT put dev paths in `.env` — they break prod.
- **Vite `host: '0.0.0.0'`**: Without this, Vite binds to `127.0.0.1` only and port forwarding from the hotspot IP won't reach it
- **Port forwarding is per-reboot**: WSL2 gets a new IP on every Windows restart. The script must be re-run each time.
- **Firewall rule cleanup**: The script deletes any legacy "Trove HTTPS" rule and recreates a unified "Trove WSL2 Ports" rule covering all three ports

---

## Phase 4: Frontend PWA Conversion

**Goal**: Make the app installable and able to load offline (app shell only — data comes in Phase 5).

### 4.1 Add dependencies
- `vite-plugin-pwa` (dev dep)

### 4.2 Self-host fonts
- Download Lora and DM Sans woff2 files → `ui/public/fonts/`
- Replace Google Fonts `<link>` tags in `index.html` with local `@font-face` declarations in CSS
- This ensures fonts work offline

### 4.3 Create PWA icons
- `ui/public/icon-192.png`, `ui/public/icon-512.png`

### 4.4 Modify `ui/vite.config.ts`
- Add `VitePWA` plugin with:
  - `registerType: 'autoUpdate'`
  - NetworkFirst for HTML, CacheFirst for JS/CSS chunks and `/uploads/*`
  - Do NOT cache `/api/*` routes (sync engine handles those)

### 4.5 Modify `ui/index.html`
- Add `<link rel="manifest">` and iOS meta tags (`apple-mobile-web-app-capable`, etc.)

### Verification
- `bun run build` produces `sw.js` in dist
- DevTools → Application → Service Workers shows registered
- Toggle offline in DevTools → app shell loads (empty data expected)
- On phone, "Add to Home Screen" works

### Gotchas
- iOS Safari: no install prompt — user must manually "Add to Home Screen" from share sheet
- SW update strategy: `autoUpdate` checks on navigation, auto-activates — safest for personal use

---

## Phase 5: IndexedDB + Offline Sync Engine

**Goal**: Replace direct API calls with local IndexedDB reads/writes. Background sync engine pushes/pulls to server. This is the core of the offline-first architecture.

### 5.1 Add dependencies
- `dexie`, `dexie-react-hooks`, `uuid` (v9+ for UUIDv7)

### 5.2 New: `ui/src/lib/db.ts`
- Dexie database `TroveDB` mirroring all 13 tables + `syncOutbox` + `syncMeta`
- Indexes: `id` on all entity tables, compound keys on junction tables, `updatedAt` on entity tables

### 5.3 New: `ui/src/lib/operations.ts`
- Local CRUD functions that write to IndexedDB + enqueue to `syncOutbox`
- Use `uuid.v7()` for client-generated IDs
- Each write sets `updatedAt = now()`, `deleted = false`
- Delete operations set `deleted = true` + update `updatedAt`
- Junction table writes bundled with parent entity

### 5.4 New: `ui/src/lib/sync-engine.ts`
- `push()`: Read outbox → build topological ChangeSet → call generated `SyncService.pushChanges()` via ConnectRPC client → clear outbox → store `serverTime`
- `pull()`: Call `SyncService.pullChanges({ since: lastSync })` → `db.table.bulkPut()` for each entity → update `lastSync`
- `sync()`: push then pull, return status
- LWW conflict resolution handled server-side
- Uses the same `rpc()` helper from `src/services/client.ts` that all existing services use — no separate fetch logic

### 5.5 New: `ui/src/lib/sync-scheduler.ts`
- `setInterval` every 10s when online, exponential backoff on failure
- Health check `/api/health` as connectivity test
- `BroadcastChannel` for multi-tab leader election (only one tab syncs)
- Expose `syncNow()` for manual trigger

### 5.6 New: `ui/src/contexts/SyncContext.tsx`
- React context providing sync state: `syncing`, `lastSyncTime`, `pendingChanges`, `online`, `syncNow()`
- Initializes sync scheduler on mount

### 5.7 New: `ui/src/hooks/useSync.ts`
- Hook consuming SyncContext

### 5.8 Rewrite all data hooks to use IndexedDB
- **`useBooks.ts`**: Replace API calls with `useLiveQuery(db.books.filter(b => !b.deleted))`. Writes go through `operations.createBook()` etc.
- **`useGames.ts`**: Same pattern
- **`useTravel.ts`**: Same pattern
- **`useWorkouts.ts`**: Same pattern
- **`useTags.ts`**: Same pattern

### 5.9 Modify `ui/src/types/api.ts`
- Add `deleted` field to all entity types
- Add `SyncOutboxEntry` type

### 5.10 Modify `ui/src/App.tsx` / `ui/src/main.tsx`
- Wrap app with `<SyncProvider>`

### 5.11 Initial hydration
- On first load (empty IndexedDB), full pull (`since=epoch`) populates all data
- Show loading indicator during bootstrap

### Verification
- Start app → initial sync fills IndexedDB (check DevTools → Application → IndexedDB)
- Create book → appears instantly, shows in outbox
- Wait for sync interval → outbox drains, book in Postgres
- Toggle offline → create book → appears in UI
- Go online → sync fires → book reaches server
- Two tabs → create in tab 1 → appears in tab 2 via `useLiveQuery`

### Gotchas
- Junction table sync: push must include junction rows for any parent in outbox
- Initial full sync could be slow with lots of data — paginate if needed
- `BroadcastChannel` leader election: if leader tab closes mid-sync, next tab takes over

---

## Phase 6: Image Sync

**Goal**: Images uploaded on one device sync to the other. Images are immutable (new UUID per upload).

### 6.1 New: `ui/src/lib/hash.ts`
- `sha256(blob: Blob): Promise<string>` using Web Crypto API

### 6.2 New: `ui/src/lib/image-sync.ts`
- **Push**: After JSON sync, query `imageBlobs` where `uploaded = false` → upload each via `POST /sync/images` → mark uploaded
- **Pull**: Compare local `imageBlobs` against server manifest (`GET /sync/images/manifest`) → download missing → store in IndexedDB

### 6.3 Update `ui/src/lib/db.ts`
- Add `imageBlobs` table: `id, checksum, uploaded`

### 6.4 Modify `ui/src/services/upload.ts`
- Now: generate UUID, compute SHA-256, store blob in IndexedDB, return local `URL.createObjectURL()`, enqueue for sync
- Upload to server happens in background via image-sync, not synchronously

### 6.5 New: `ui/src/hooks/useImageUrl.ts`
- Check IndexedDB first for blob → `createObjectURL()`, fall back to server URL
- Revoke object URLs on unmount to prevent memory leaks

### 6.6 Modify `ui/src/lib/sync-engine.ts`
- After JSON push/pull, call `imageSync.push()` and `imageSync.pull()`

### 6.7 Update image display in components
- Books/Games cover image rendering uses `useImageUrl(path)` hook

### Verification
- Upload image on laptop → syncs to phone's IndexedDB
- Upload image on phone offline → appears locally via object URL
- Connect → image uploads to server → available on laptop
- SHA-256 checksum validation works

### Gotchas
- Mobile Safari IndexedDB limit ~1GB — fine for personal use, add cleanup for old deleted blobs
- Upload images sequentially (not parallel) over hotspot Wi-Fi
- `URL.revokeObjectURL()` on unmount to prevent memory leaks

---

## Phase 7: UI Indicators & Polish

**Goal**: Visual feedback for sync status, offline mode, and pending changes.

### 7.1 New: `ui/src/components/SyncIndicator.tsx`
- Persistent icon in navbar:
  - Green cloud: "Synced" (outbox empty, recent sync)
  - Yellow dot: "Pending changes" (outbox has entries)
  - Spinner: "Syncing..."
  - Red triangle: "Sync error"
  - Gray: "Offline"
- Click → popover with details, manual "Sync Now" button

### 7.2 New: `ui/src/components/OfflineBanner.tsx`
- Thin banner when offline: "Working offline — changes will sync when connected"

### 7.3 Modify `ui/src/components/Layout.tsx`
- Add `<SyncIndicator />` to navbar

### Verification
- Connected + idle → green
- Create entry → yellow → spinning → green
- Toggle offline → gray + banner
- Create offline → yellow badge with count
- Go online → spinning → green

---

## Phase Dependency Graph

```
Phase 1 (DB Schema) → Phase 2 (Sync API) → Phase 5 (IndexedDB + Sync)
                                                ↓
Phase 3 (TLS) — can be done anytime        Phase 6 (Image Sync)
Phase 4 (PWA) — can be done anytime            ↓
                                           Phase 7 (UI Polish)
```

**Recommended order**: 1 → 2 → 4 → 3 → 5 → 6 → 7

Do PWA shell (4) early — low risk, immediate value (installable app).
Do TLS (3) before Phase 5 — phone needs HTTPS to reach sync endpoints.

---

## New Dependencies Summary

| Layer | Package | Purpose |
|-------|---------|---------|
| Frontend | `dexie` | IndexedDB wrapper |
| Frontend | `dexie-react-hooks` | `useLiveQuery` for reactive reads |
| Frontend | `uuid` (v9+) | UUIDv7 client-side ID generation |
| Frontend (dev) | `vite-plugin-pwa` | Service worker + manifest generation |
| System | `mkcert` | Local TLS certificate generation |

No new Go dependencies needed — `crypto/sha256` is in stdlib.

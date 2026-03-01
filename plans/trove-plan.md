# Trove — Design & Implementation Plan

This application acts like a library of life experiences. It contains four sections: Books, Video Games, Travel, and Workouts.

---

## Functional Requirements

### Books

* Will have title, author, years read in, rating out of 5, optional review, category tags, and a cover image
* Books can be grouped into collections. Collection will have name and list of books in it. Collections can be made based on author, tags, years read in or books can be manually added to a collection. A book can belong to multiple collections
* UI will
  * Display all books as a grid or list and provide option to toggle between the two viewing modes
  * Way to add/update book info/delete a book
  * Way to organise the books into a collection
  * Search by book name, author name, tag name, collection name, years read in

### Video Games

* Will have title, studio/publisher name, rating, years played in, cover image, optional review
* No collections or tags required for video games
* UI will
  * Separate page to show all video games in a grid or list
  * Way to add/update/delete an item
  * Sorting and Searching

### Travel

* Will have location details (city, country), list of tourist spots visited (each with optional description), dates when visited, and links to photograph collection
* UI will
  * Separate page to show grid or list of locations
  * Clicking on an item will take to the photograph collection
  * Searching by location

### Workout

* Weekly status tracking of particular workout in a tabular form, for example:

  | Week | Exercise | Sets | Reps/Duration | Weight (kg) |
  |------|----------|------|---------------|-------------|
  | 2 | Push Ups | 5 | 10 | |
  | | Flat Bench Press | 3 | 12 | 25 |
  | | Incline D/B press | 3 | 10 | 12 |
  | 1 | Push Ups | 5 | 10 | |
  | | Flat Bench Press | 3 | 10 | 25 |

* Way to add in the progress information
* Graph views wherever required

### Common Functional Requirements

* UI should have light/dark mode toggle and consistent styling and design system throughout

---

## Key Technical Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Database | PostgreSQL | Relational data with collections, tags, many-to-many relationships, and weekly tracking. JSONB available for flexible fields. |
| Image storage | Local filesystem | Images stored on disk in a mounted volume, served via the Go backend. Simple, no extra infra. |
| gRPC-to-browser | ConnectRPC (connect-go + connect-web) | Native gRPC-Web support in Go without a proxy like Envoy. Cleaner stack. |
| Containerization | Podman + podman-compose | Per the project design document. |

---

## Directory Structure

```
life-management-system/
├── ui/                        # React + Bun.js frontend
│   ├── src/
│   │   ├── components/        # Reusable UI components
│   │   ├── pages/             # Page-level components (Books, Games, Travel, Workouts)
│   │   ├── hooks/             # Custom React hooks
│   │   ├── services/          # ConnectRPC generated clients
│   │   ├── theme/             # Chakra UI theme config (colors, tokens)
│   │   └── types/             # Shared TypeScript types
│   ├── public/
│   ├── package.json
│   └── bunfig.toml
├── backend/
│   ├── cmd/server/            # Main entrypoint
│   ├── internal/
│   │   ├── controller/        # gRPC/Connect handlers
│   │   ├── service/           # Business logic layer
│   │   ├── dao/               # Data access layer (SQL queries)
│   │   ├── model/             # Go structs mapping to DB tables
│   │   └── config/            # App configuration
│   ├── proto/                 # Protobuf definitions
│   ├── migrations/            # SQL migration files
│   ├── go.mod
│   └── go.sum
├── infra/
│   ├── podman-compose.yml     # Full stack: UI, backend, Postgres, Redis
│   ├── Containerfile.ui
│   ├── Containerfile.backend
│   └── init-db.sql            # Initial DB setup
├── uploads/                   # Mounted volume for images (gitignored)
└── plans/                     # Architecture docs
```

---

## Database Schema (PostgreSQL)

### Books

```sql
CREATE TABLE books (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(500) NOT NULL,
    author      VARCHAR(300) NOT NULL,
    rating      SMALLINT CHECK (rating BETWEEN 1 AND 5),
    review      TEXT,
    cover_image VARCHAR(500),          -- relative path in uploads/
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE book_years_read (
    book_id UUID REFERENCES books(id) ON DELETE CASCADE,
    year    SMALLINT NOT NULL,
    PRIMARY KEY (book_id, year)
);

CREATE TABLE tags (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE book_tags (
    book_id UUID REFERENCES books(id) ON DELETE CASCADE,
    tag_id  UUID REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (book_id, tag_id)
);

CREATE TABLE collections (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(300) UNIQUE NOT NULL
);

CREATE TABLE collection_books (
    collection_id UUID REFERENCES collections(id) ON DELETE CASCADE,
    book_id       UUID REFERENCES books(id) ON DELETE CASCADE,
    PRIMARY KEY (collection_id, book_id)
);
```

### Video Games

```sql
CREATE TABLE video_games (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(500) NOT NULL,
    studio      VARCHAR(300),
    rating      SMALLINT CHECK (rating BETWEEN 1 AND 5),
    review      TEXT,
    cover_image VARCHAR(500),
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE game_years_played (
    game_id UUID REFERENCES video_games(id) ON DELETE CASCADE,
    year    SMALLINT NOT NULL,
    PRIMARY KEY (game_id, year)
);
```

### Travel

```sql
CREATE TABLE travel_locations (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city                 VARCHAR(300) NOT NULL,
    country              VARCHAR(300) NOT NULL,
    visited_from         DATE,
    visited_to           DATE,
    photo_collection_url VARCHAR(500),  -- link to photography portfolio or external album
    created_at           TIMESTAMPTZ DEFAULT now(),
    updated_at           TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE tourist_spots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id UUID REFERENCES travel_locations(id) ON DELETE CASCADE,
    name        VARCHAR(300) NOT NULL,
    description TEXT
);
```

### Workouts

Workout types are dynamic — users define their own types (Push Day, Pull Day, Legs, etc.).

```sql
CREATE TABLE workout_types (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(200) NOT NULL,          -- e.g. "Push Day", "Pull Day"
    sort_order SMALLINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE exercises (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_type_id UUID REFERENCES workout_types(id) ON DELETE CASCADE,
    name            VARCHAR(300) NOT NULL,      -- e.g. "Flat Bench Press"
    sort_order      SMALLINT DEFAULT 0
);

CREATE TABLE workout_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exercise_id UUID REFERENCES exercises(id) ON DELETE CASCADE,
    week_number SMALLINT NOT NULL,
    sets        SMALLINT,
    reps        VARCHAR(50),                    -- "10" or "30s" for duration-based
    weight_kg   DECIMAL(6,2),
    logged_at   TIMESTAMPTZ DEFAULT now()
);
```

### Indexes

```sql
CREATE INDEX idx_books_title ON books (title);
CREATE INDEX idx_books_author ON books (author);
CREATE INDEX idx_video_games_title ON video_games (title);
CREATE INDEX idx_travel_city ON travel_locations (city);
CREATE INDEX idx_travel_country ON travel_locations (country);
CREATE INDEX idx_workout_logs_exercise ON workout_logs (exercise_id, week_number);
```

---

## Implementation Phases

### Phase 1: Project Scaffolding & Infrastructure

**Goal:** Set up the skeleton so every subsequent phase starts with a working, deployable stack.

1. Create the `life-management-system/` directory structure as shown above
2. Initialize Go module (`backend/`) with connect-go dependencies
3. Initialize React project (`ui/`) with Bun, Chakra UI, and connect-web
4. Write Containerfiles for UI and backend
5. Write `podman-compose.yml` bringing up: Postgres, Redis, Go backend, React UI
6. Write SQL migrations for the full schema above
7. Set up Chakra UI theme with flexible color tokens (light/dark mode)
8. Verify the full stack starts with a single `podman-compose up`

### Phase 2: Backend — Core API

**Goal:** Implement all protobuf definitions and backend CRUD operations.

1. **Define protobuf schemas** in `backend/proto/` for all four sections:
   - `books.proto` — CRUD for books, tags, collections, collection membership
   - `games.proto` — CRUD for video games
   - `travel.proto` — CRUD for locations and tourist spots
   - `workouts.proto` — CRUD for workout types, exercises, workout logs
   - `common.proto` — Shared messages (pagination, search filters, image upload)

2. **Generate Go and TypeScript code** from proto definitions

3. **Implement backend layers** (for each section):
   - `dao/` — SQL queries using `pgx` (Go Postgres driver)
   - `service/` — Business logic, validation
   - `controller/` — Connect handlers delegating to service layer

4. **Image upload endpoint** — Accept multipart upload, store to `uploads/` volume, return relative path

5. **Search** — Implement keyword search across relevant fields per section (book title/author/tag, game title/studio, location city/country)

6. **Redis caching** — Cache frequently read lists (all books, all games) with TTL-based invalidation on writes

### Phase 3: Frontend — Books Section

**Goal:** Build the most feature-rich section first to establish all reusable UI patterns.

1. **Reusable components:**
   - `ItemCard` — Grid card with cover image, title, subtitle, rating stars
   - `ItemList` — List-view row variant of the same data
   - `ViewToggle` — Grid/list toggle control
   - `SearchBar` — Debounced search input
   - `RatingDisplay` / `RatingInput` — Star rating display and selector
   - `ImageUpload` — Drag-and-drop cover image uploader
   - `ConfirmDialog` — Reusable delete confirmation
   - `ThemeToggle` — Light/dark mode switch

2. **Books page:**
   - Grid/list view of all books with cover, title, author, rating
   - Add/edit book modal/form
   - Delete with confirmation
   - Tag management (assign/remove tags)
   - Collection management (create collections, add/remove books)
   - Search by name, author, tag, collection, year

### Phase 4: Frontend — Video Games & Travel

**Goal:** Reuse components from Phase 3 for the remaining media sections.

1. **Video Games page:**
   - Grid/list view reusing `ItemCard` / `ItemList`
   - Add/edit/delete game
   - Sort by title, rating, year
   - Search by title, studio

2. **Travel page:**
   - Grid/list of locations reusing shared components
   - Click-through to photo collection (external link)
   - Location detail view showing tourist spots
   - Add/edit/delete locations and spots
   - Search by city, country

### Phase 5: Frontend — Workouts

**Goal:** Build the workout tracker with tabular input and graph views.

1. **Workout types management** — Add/edit/delete workout types (Push Day, Pull Day, etc.)
2. **Exercise management** — Define exercises per workout type with ordering
3. **Weekly log entry** — Tabular form to log sets/reps/weight per exercise per week
4. **Week-over-week table view** — Display grouped by week as shown in the requirements
5. **Progress graphs** — Line charts showing weight/reps progression over weeks per exercise (using a charting library like Recharts)

### Phase 6: Polish & Integration

**Goal:** Final touches across the whole application.

1. Light/dark mode working consistently across all pages
2. Navigation — sidebar or top nav linking all four sections
3. Responsive layout for different screen sizes
4. Loading states and error handling
5. Empty states (no books yet, no workouts logged, etc.)

---

## Verification Plan

After each phase, verify:

1. **Phase 1:** `podman-compose up` brings up all containers; Postgres is reachable; React dev server loads
2. **Phase 2:** All gRPC endpoints respond correctly via `buf curl` or connect-web test client; image upload/retrieval works; Redis caching works
3. **Phase 3-5:** Each page renders correctly; CRUD operations work end-to-end through the UI; search returns expected results
4. **Phase 6:** Toggle dark/light mode on every page; test on different viewport sizes; verify no console errors

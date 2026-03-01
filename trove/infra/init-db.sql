-- Books
CREATE TABLE IF NOT EXISTS books (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(500) NOT NULL,
    author      VARCHAR(300) NOT NULL,
    rating      SMALLINT CHECK (rating BETWEEN 1 AND 5),
    review      TEXT,
    cover_image VARCHAR(500),
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS book_years_read (
    book_id UUID REFERENCES books(id) ON DELETE CASCADE,
    year    SMALLINT NOT NULL,
    PRIMARY KEY (book_id, year)
);

CREATE TABLE IF NOT EXISTS tags (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS book_tags (
    book_id UUID REFERENCES books(id) ON DELETE CASCADE,
    tag_id  UUID REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (book_id, tag_id)
);

CREATE TABLE IF NOT EXISTS collections (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(300) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS collection_books (
    collection_id UUID REFERENCES collections(id) ON DELETE CASCADE,
    book_id       UUID REFERENCES books(id) ON DELETE CASCADE,
    PRIMARY KEY (collection_id, book_id)
);

-- Video Games
CREATE TABLE IF NOT EXISTS video_games (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(500) NOT NULL,
    studio      VARCHAR(300),
    rating      SMALLINT CHECK (rating BETWEEN 1 AND 5),
    review      TEXT,
    cover_image VARCHAR(500),
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game_years_played (
    game_id UUID REFERENCES video_games(id) ON DELETE CASCADE,
    year    SMALLINT NOT NULL,
    PRIMARY KEY (game_id, year)
);

-- Travel
CREATE TABLE IF NOT EXISTS travel_locations (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city                 VARCHAR(300) NOT NULL,
    country              VARCHAR(300) NOT NULL,
    visited_from         DATE,
    visited_to           DATE,
    photo_collection_url VARCHAR(500),
    created_at           TIMESTAMPTZ DEFAULT now(),
    updated_at           TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tourist_spots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id UUID REFERENCES travel_locations(id) ON DELETE CASCADE,
    name        VARCHAR(300) NOT NULL,
    description TEXT
);

-- Workouts
CREATE TABLE IF NOT EXISTS workout_types (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(200) NOT NULL,
    sort_order SMALLINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS exercises (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workout_type_id UUID REFERENCES workout_types(id) ON DELETE CASCADE,
    name            VARCHAR(300) NOT NULL,
    sort_order      SMALLINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS workout_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exercise_id UUID REFERENCES exercises(id) ON DELETE CASCADE,
    week_number SMALLINT NOT NULL,
    sets        SMALLINT,
    reps        VARCHAR(50),
    weight_kg   DECIMAL(6,2),
    logged_at   TIMESTAMPTZ DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_books_title           ON books (title);
CREATE INDEX IF NOT EXISTS idx_books_author          ON books (author);
CREATE INDEX IF NOT EXISTS idx_video_games_title     ON video_games (title);
CREATE INDEX IF NOT EXISTS idx_travel_city           ON travel_locations (city);
CREATE INDEX IF NOT EXISTS idx_travel_country        ON travel_locations (country);
CREATE INDEX IF NOT EXISTS idx_workout_logs_exercise ON workout_logs (exercise_id, week_number);

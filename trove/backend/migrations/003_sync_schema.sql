-- Add updated_at to tables that don't have it yet
ALTER TABLE tags ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE collections ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE tourist_spots ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE workout_types ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE workout_logs ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();

-- Add deleted flag to all entity tables
ALTER TABLE books ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE video_games ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE travel_locations ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE tags ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE collections ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE tourist_spots ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE workout_types ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE workout_logs ADD COLUMN IF NOT EXISTS deleted BOOLEAN NOT NULL DEFAULT false;

-- Indexes on updated_at for efficient delta sync queries
CREATE INDEX IF NOT EXISTS idx_books_updated_at ON books (updated_at);
CREATE INDEX IF NOT EXISTS idx_video_games_updated_at ON video_games (updated_at);
CREATE INDEX IF NOT EXISTS idx_travel_locations_updated_at ON travel_locations (updated_at);
CREATE INDEX IF NOT EXISTS idx_tags_updated_at ON tags (updated_at);
CREATE INDEX IF NOT EXISTS idx_collections_updated_at ON collections (updated_at);
CREATE INDEX IF NOT EXISTS idx_tourist_spots_updated_at ON tourist_spots (updated_at);
CREATE INDEX IF NOT EXISTS idx_workout_types_updated_at ON workout_types (updated_at);
CREATE INDEX IF NOT EXISTS idx_exercises_updated_at ON exercises (updated_at);
CREATE INDEX IF NOT EXISTS idx_workout_logs_updated_at ON workout_logs (updated_at);

-- Sync metadata table for server-side sync state
CREATE TABLE IF NOT EXISTS sync_metadata (
    key   VARCHAR(100) PRIMARY KEY,
    value TEXT
);

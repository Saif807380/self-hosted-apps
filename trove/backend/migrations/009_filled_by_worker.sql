-- The cover and studio the worker filled in from a game's IGDB match. When
-- you relink the game to a different IGDB game, values that are still the
-- filled ones are cleared and refilled from the new match; a cover or studio
-- you set yourself is kept.
ALTER TABLE game_meta ADD COLUMN IF NOT EXISTS filled_cover  TEXT;
ALTER TABLE game_meta ADD COLUMN IF NOT EXISTS filled_studio TEXT;

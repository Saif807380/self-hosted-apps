-- One-off: games linked to IGDB automatically before the worker started
-- renaming them take IGDB's title, as picked matches always have. updated_at
-- moves so the new names reach every device on its next sync. From now on the
-- worker renames a game once, when it first links it.
UPDATE video_games g
SET title = m.title, updated_at = now()
FROM game_meta m
WHERE m.video_game_id = g.id
  AND m.match_method = 'auto'
  AND m.title IS NOT NULL AND m.title <> ''
  AND g.title <> m.title
  AND g.deleted = false;

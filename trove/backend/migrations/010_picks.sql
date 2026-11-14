-- Weekly picks: games on sale that you'd probably like. Written by the
-- worker each Sunday; you act on them from the Deals tab.
CREATE TABLE IF NOT EXISTS picks (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    week          DATE NOT NULL,          -- the Sunday the picks were made for (IST)
    rank          SMALLINT NOT NULL,      -- 1 = the model's first choice
    igdb_id       BIGINT NOT NULL,
    title         VARCHAR(500) NOT NULL,
    year          SMALLINT,
    cover_url     VARCHAR(500),
    pitch         TEXT NOT NULL,
    tags          TEXT[] NOT NULL DEFAULT '{}', -- a dismissed pick counts against these
    store         VARCHAR(16) NOT NULL CHECK (store IN ('steam', 'eshop')),
    store_ref     VARCHAR(100) NOT NULL,
    edition       VARCHAR(50) NOT NULL DEFAULT '',
    url           VARCHAR(1000) NOT NULL,
    currency      VARCHAR(8) NOT NULL,
    base_minor    BIGINT NOT NULL,
    final_minor   BIGINT NOT NULL,
    discount_pct  SMALLINT NOT NULL,
    sale_ends_at  TIMESTAMPTZ,
    state         VARCHAR(16) NOT NULL DEFAULT 'new' CHECK (state IN ('new', 'wishlisted', 'owned', 'dismissed')),
    video_game_id UUID REFERENCES video_games(id) ON DELETE SET NULL, -- the game made from it
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (week, igdb_id)
);
CREATE INDEX IF NOT EXISTS idx_picks_week ON picks (week);

-- One row per weekly run, even one that found nothing, so a week isn't
-- retried every hour.
CREATE TABLE IF NOT EXISTS pick_runs (
    week   DATE PRIMARY KEY,
    picks  SMALLINT NOT NULL,
    ran_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

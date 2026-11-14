-- Price tracking. Everything here is written by the worker and read by the
-- server; none of it is part of offline sync, so the worker never touches
-- video_games and can't conflict with an offline edit.

-- IGDB identity and tags for a game. Every game gets one (played games too,
-- for recommendations later); unresolved rows hold candidates for the UI.
CREATE TABLE IF NOT EXISTS game_meta (
    video_game_id UUID PRIMARY KEY REFERENCES video_games(id) ON DELETE CASCADE,
    match_method  VARCHAR(16) NOT NULL CHECK (match_method IN ('picked', 'auto', 'unresolved')),
    igdb_id       BIGINT,
    title         VARCHAR(500),
    release_year  SMALLINT,
    cover_url     VARCHAR(500),
    steam_appid   VARCHAR(32),
    platforms     INTEGER[] NOT NULL DEFAULT '{}',
    developers    TEXT[]    NOT NULL DEFAULT '{}',
    publishers    TEXT[]    NOT NULL DEFAULT '{}',
    genres        TEXT[]    NOT NULL DEFAULT '{}',
    themes        TEXT[]    NOT NULL DEFAULT '{}',
    keywords      TEXT[]    NOT NULL DEFAULT '{}',
    igdb_rating   REAL,
    similar_ids   BIGINT[]  NOT NULL DEFAULT '{}',
    candidates    JSONB,    -- unresolved only: [{igdbId, title, year, coverUrl}]
    listings_at   TIMESTAMPTZ, -- when store listings were last looked up
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (match_method = 'unresolved' OR igdb_id IS NOT NULL)
);

-- One purchasable version of a game in one store.
CREATE TABLE IF NOT EXISTS store_listings (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_game_id UUID NOT NULL REFERENCES video_games(id) ON DELETE CASCADE,
    store         VARCHAR(16) NOT NULL CHECK (store IN ('steam', 'eshop', 'psn')),
    store_ref     VARCHAR(100) NOT NULL DEFAULT '', -- '' = PS Store search link
    edition       VARCHAR(50) NOT NULL DEFAULT '',
    title         VARCHAR(500) NOT NULL,
    url           VARCHAR(1000) NOT NULL,
    region        VARCHAR(8) NOT NULL,
    currency      VARCHAR(8) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (video_game_id, store, store_ref)
);
CREATE INDEX IF NOT EXISTS idx_store_listings_game ON store_listings (video_game_id);

-- Price history as runs of an unchanged price: a new row only when the price
-- changes, otherwise last_seen_at moves forward. Amounts in minor units.
CREATE TABLE IF NOT EXISTS price_quotes (
    id            BIGSERIAL PRIMARY KEY,
    listing_id    UUID NOT NULL REFERENCES store_listings(id) ON DELETE CASCADE,
    base_minor    BIGINT NOT NULL,
    final_minor   BIGINT NOT NULL,
    discount_pct  SMALLINT NOT NULL DEFAULT 0,
    sale_ends_at  TIMESTAMPTZ,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_price_quotes_listing ON price_quotes (listing_id, first_seen_at);

-- Sales worth telling you about. An outbox: the push sender delivers rows
-- with notified_at NULL. sale_key makes each sale alert at most once.
CREATE TABLE IF NOT EXISTS sale_events (
    id          BIGSERIAL PRIMARY KEY,
    listing_id  UUID NOT NULL REFERENCES store_listings(id) ON DELETE CASCADE,
    sale_key    VARCHAR(100) NOT NULL,
    reasons     TEXT[] NOT NULL,
    final_minor BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    notified_at TIMESTAMPTZ,
    UNIQUE (listing_id, sale_key)
);
CREATE INDEX IF NOT EXISTS idx_sale_events_pending ON sale_events (created_at) WHERE notified_at IS NULL;

-- Latest exchange rates, e.g. USD->INR for comparing eShop prices.
CREATE TABLE IF NOT EXISTS fx_rates (
    base       VARCHAR(8) NOT NULL,
    quote      VARCHAR(8) NOT NULL,
    rate       DOUBLE PRECISION NOT NULL CHECK (rate > 0),
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (base, quote)
);

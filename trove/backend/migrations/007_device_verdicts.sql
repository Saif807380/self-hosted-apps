-- Device verdicts: which of your devices (the G15 laptop, PS5, Switch 2) to
-- play a wishlisted game on. Written by the worker from gathered evidence and
-- one Claude call; never synced to clients.
CREATE TABLE IF NOT EXISTS device_verdicts (
    video_game_id UUID PRIMARY KEY REFERENCES video_games(id) ON DELETE CASCADE,
    verdict       JSONB NOT NULL,       -- fit.Verdict: summary, ranked devices, unavailable
    sources       JSONB NOT NULL,       -- [{label, url}] the evidence came from
    model         VARCHAR(64) NOT NULL, -- the model that answered (a refusal fallback may differ)
    complete      BOOLEAN NOT NULL,     -- false: some evidence was missing; redone once it's there
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    checked_at    TIMESTAMPTZ NOT NULL DEFAULT now() -- last time an incomplete verdict was retried
);

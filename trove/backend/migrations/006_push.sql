-- Web push subscriptions: one per browser that turned on sale alerts (the
-- Trove PWA on your phone, a desktop browser). Removed when the push service
-- reports the subscription gone.
CREATE TABLE IF NOT EXISTS push_subscriptions (
    endpoint     VARCHAR(1000) PRIMARY KEY,
    p256dh       VARCHAR(200) NOT NULL,
    auth         VARCHAR(100) NOT NULL,
    user_agent   VARCHAR(500),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_ok_at   TIMESTAMPTZ
);

-- What a sale notification needs to say ("−30%, was ₹1,300"). Events
-- recorded before this migration have neither.
ALTER TABLE sale_events ADD COLUMN IF NOT EXISTS base_minor BIGINT;
ALTER TABLE sale_events ADD COLUMN IF NOT EXISTS discount_pct SMALLINT;

-- Game lifecycle: a video_games row is now any game you care about, not only
-- ones you've played. Existing rows are all played games, hence the default.
ALTER TABLE video_games ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'played';
ALTER TABLE video_games ADD COLUMN IF NOT EXISTS platform VARCHAR(16);
-- Alert threshold for a wishlisted game, in whole rupees. eShop USD prices are
-- converted to INR before comparing.
ALTER TABLE video_games ADD COLUMN IF NOT EXISTS target_price_inr INTEGER;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'video_games_status_check') THEN
        ALTER TABLE video_games ADD CONSTRAINT video_games_status_check
            CHECK (status IN ('wishlist', 'owned', 'played', 'dropped'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'video_games_platform_check') THEN
        ALTER TABLE video_games ADD CONSTRAINT video_games_platform_check
            CHECK (platform IS NULL OR platform IN ('pc', 'ps5', 'switch2'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'video_games_target_price_check') THEN
        ALTER TABLE video_games ADD CONSTRAINT video_games_target_price_check
            CHECK (target_price_inr IS NULL OR target_price_inr > 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_video_games_status ON video_games (status);

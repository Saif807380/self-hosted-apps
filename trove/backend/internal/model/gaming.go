package model

import (
	"time"

	"github.com/google/uuid"
)

// Match methods for GameMeta: how a Trove game was tied to an IGDB game.
const (
	MatchPicked     = "picked"     // you chose it
	MatchAuto       = "auto"       // linked automatically (catalog.ChooseMatch)
	MatchUnresolved = "unresolved" // ambiguous; Candidates holds the options
)

// GameMeta is a game's IGDB identity and tags. Written by the worker (or by
// LinkGame), never synced to clients.
type GameMeta struct {
	VideoGameID uuid.UUID        `db:"video_game_id"`
	MatchMethod string           `db:"match_method"`
	IGDBID      *int64           `db:"igdb_id"`
	Title       *string          `db:"title"`
	ReleaseYear *int16           `db:"release_year"`
	CoverURL    *string          `db:"cover_url"`
	SteamAppID  *string          `db:"steam_appid"`
	Platforms   []int32          `db:"platforms"`
	Developers  []string         `db:"developers"`
	Publishers  []string         `db:"publishers"`
	Genres      []string         `db:"genres"`
	Themes      []string         `db:"themes"`
	Keywords    []string         `db:"keywords"`
	IGDBRating  *float32         `db:"igdb_rating"`
	SimilarIDs  []int64          `db:"similar_ids"`
	Candidates  []MatchCandidate `db:"candidates"`
	ListingsAt  *time.Time       `db:"listings_at"`
	UpdatedAt   time.Time        `db:"updated_at"`
}

// MatchCandidate is one IGDB game offered when a match is ambiguous.
type MatchCandidate struct {
	IGDBID   int64  `json:"igdbId"`
	Title    string `json:"title"`
	Year     int    `json:"year,omitempty"`
	CoverURL string `json:"coverUrl,omitempty"`
}

// StoreListing is one purchasable version of a game in one store.
type StoreListing struct {
	ID          uuid.UUID `db:"id"`
	VideoGameID uuid.UUID `db:"video_game_id"`
	Store       string    `db:"store"`
	StoreRef    string    `db:"store_ref"`
	Edition     string    `db:"edition"`
	Title       string    `db:"title"`
	URL         string    `db:"url"`
	Region      string    `db:"region"`
	Currency    string    `db:"currency"`
	CreatedAt   time.Time `db:"created_at"`
}

// PriceRun is a stretch of time a listing held one price.
type PriceRun struct {
	ID          int64      `db:"id"`
	ListingID   uuid.UUID  `db:"listing_id"`
	BaseMinor   int64      `db:"base_minor"`
	FinalMinor  int64      `db:"final_minor"`
	DiscountPct int16      `db:"discount_pct"`
	SaleEndsAt  *time.Time `db:"sale_ends_at"`
	FirstSeenAt time.Time  `db:"first_seen_at"`
	LastSeenAt  time.Time  `db:"last_seen_at"`
}

package dao

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/stores"
)

// pricedGame adds a game with one Steam listing and records the given
// final prices against a ₹1,000 base, oldest first. It returns the game.
func pricedGame(t *testing.T, s *Stores, status string, finals ...int64) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	g, err := s.Games.Create(ctx, CreateGameParams{Title: "Deal " + status, Status: status, YearsPlayed: []int16{}})
	if err != nil {
		t.Fatal(err)
	}
	listing := stores.Listing{Store: stores.Steam, Ref: uuid.NewString()[:8], Title: g.Title, URL: "https://example.com", Region: "IN", Currency: "INR"}
	if err := s.Gaming.SaveListings(ctx, g.ID, []stores.Listing{listing}); err != nil {
		t.Fatal(err)
	}
	listings, err := s.Gaming.Listings(ctx, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, final := range finals {
		q := stores.Quote{Currency: "INR", BaseMinor: 100000, FinalMinor: final, CheckedAt: time.Now()}
		if _, _, err := s.Gaming.RecordQuote(ctx, listings[0].ID, q); err != nil {
			t.Fatal(err)
		}
	}
	return g.ID
}

func TestListDeals(t *testing.T) {
	ctx := context.Background()
	s := testStores(t)

	deeper := pricedGame(t, s, model.GameWishlist, 100000, 50000) // −50%, lowest yet
	back := pricedGame(t, s, model.GameWishlist, 40000, 100000, 70000)
	firstSeen := pricedGame(t, s, model.GameWishlist, 50000)         // on sale when first checked
	fullPrice := pricedGame(t, s, model.GameWishlist, 70000, 100000) // sale over
	played := pricedGame(t, s, model.GamePlayed, 50000)
	stale := pricedGame(t, s, model.GameWishlist, 50000)
	if _, err := s.Gaming.db.Exec(ctx, `
		UPDATE price_quotes SET last_seen_at = now() - interval '3 days'
		WHERE listing_id IN (SELECT id FROM store_listings WHERE video_game_id = $1)`, stale); err != nil {
		t.Fatal(err)
	}

	deals, err := s.Gaming.ListDeals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[uuid.UUID]Deal{}
	for _, d := range deals {
		got[d.VideoGameID] = d
	}
	if d, ok := got[deeper]; !ok || d.FinalMinor != 50000 || !d.Lowest {
		t.Errorf("deeper discount = %+v, %v", d, ok)
	}
	if d, ok := got[back]; !ok || d.FinalMinor != 70000 || d.Lowest {
		t.Errorf("sale again, not a new low = %+v, %v", d, ok)
	}
	if d, ok := got[firstSeen]; !ok || d.Lowest {
		t.Errorf("first price = %+v, %v: listed, but not as a record low", d, ok)
	}
	for name, id := range map[string]uuid.UUID{"full price": fullPrice, "played": played, "stale": stale} {
		if _, ok := got[id]; ok {
			t.Errorf("%s game listed as a deal", name)
		}
	}
}

package dao

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saifkazi/trove/backend/internal/model"
)

// testStores connects to a disposable database with every migration applied.
// It is skipped unless TROVE_TEST_DATABASE_URL is set; never point it at the
// real Trove database, because the tests insert rows.
func testStores(t *testing.T) *Stores {
	t.Helper()
	url := os.Getenv("TROVE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TROVE_TEST_DATABASE_URL not set")
	}
	pool, err := NewPostgresPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return NewStores(pool)
}

func strPtr(s string) *string { return &s }
func i32Ptr(n int32) *int32   { return &n }

func TestGameLifecycleCRUD(t *testing.T) {
	ctx := context.Background()
	s := testStores(t)

	g, err := s.Games.Create(ctx, CreateGameParams{
		Title: "Hades II " + uuid.NewString(), Status: model.GameWishlist,
		Platform: strPtr("switch2"), TargetPriceINR: i32Ptr(1500), YearsPlayed: []int16{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != model.GameWishlist || *g.Platform != "switch2" || *g.TargetPriceINR != 1500 {
		t.Fatalf("created = %+v", g)
	}

	// A client without lifecycle fields must not wipe them.
	g, err = s.Games.Update(ctx, UpdateGameParams{ID: g.ID, Title: g.Title, YearsPlayed: []int16{}})
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != model.GameWishlist || g.Platform == nil || g.TargetPriceINR == nil {
		t.Fatalf("update without lifecycle changed it: %+v", g)
	}

	// Finishing the game: status to played, target cleared.
	g, err = s.Games.Update(ctx, UpdateGameParams{
		ID: g.ID, Title: g.Title, Rating: func() *int16 { r := int16(5); return &r }(), YearsPlayed: []int16{2026},
		Lifecycle: &GameLifecycle{Status: model.GamePlayed, Platform: strPtr("switch2")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.Status != model.GamePlayed || g.TargetPriceINR != nil {
		t.Fatalf("update with lifecycle = %+v", g)
	}

	played, err := s.Games.List(ctx, GameFilter{Search: g.Title, Status: model.GamePlayed})
	if err != nil || len(played) != 1 || played[0].ID != g.ID {
		t.Fatalf("list played = %v, %v", played, err)
	}
	wishlist, err := s.Games.List(ctx, GameFilter{Search: g.Title, Status: model.GameWishlist})
	if err != nil || len(wishlist) != 0 {
		t.Fatalf("list wishlist = %v, %v", wishlist, err)
	}
}

func TestGameLifecycleSync(t *testing.T) {
	ctx := context.Background()
	s := testStores(t)
	sync := &SyncStore{db: s.DB}
	since := time.Now().UTC().Add(-time.Second)

	// New client creates a wishlist game offline.
	id := uuid.New()
	game := model.VideoGame{
		ID: id, Title: "Silksong " + id.String(), Status: model.GameWishlist,
		Platform: strPtr("pc"), TargetPriceINR: i32Ptr(700), YearsPlayed: []int16{},
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	push := func(g model.VideoGame) {
		t.Helper()
		if err := sync.ApplyChanges(ctx, &model.ChangeSet{VideoGames: []model.VideoGame{g}}); err != nil {
			t.Fatal(err)
		}
	}
	get := func() *model.VideoGame {
		t.Helper()
		g, err := s.Games.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	push(game)
	if g := get(); g.Status != model.GameWishlist || *g.TargetPriceINR != 700 {
		t.Fatalf("after new-client push: %+v", g)
	}

	// An old cached client edits the title; it knows nothing of the lifecycle.
	old := game
	old.Title += " (renamed)"
	old.Status, old.Platform, old.TargetPriceINR = "", nil, nil
	push(old)
	if g := get(); g.Title != old.Title || g.Status != model.GameWishlist || g.Platform == nil || g.TargetPriceINR == nil {
		t.Fatalf("old-client push changed lifecycle: %+v", g)
	}

	// New client marks it owned.
	owned := game
	owned.Status, owned.TargetPriceINR = model.GameOwned, nil
	push(owned)
	if g := get(); g.Status != model.GameOwned || g.TargetPriceINR != nil {
		t.Fatalf("after owned push: %+v", g)
	}

	// The worker adds a cover; a device that hasn't synced it yet pushes the
	// game without one. The cover must survive.
	if _, err := s.Gaming.SetCoverIfEmpty(ctx, id, "/uploads/auto.jpg"); err != nil {
		t.Fatal(err)
	}
	if err := s.Gaming.SetStudioIfEmpty(ctx, id, "Team Cherry"); err != nil {
		t.Fatal(err)
	}
	stale := owned
	stale.CoverImage, stale.Studio = nil, nil
	push(stale)
	if g := get(); g.CoverImage == nil || *g.CoverImage != "/uploads/auto.jpg" || g.Studio == nil || *g.Studio != "Team Cherry" {
		t.Fatalf("push without a cover/studio cleared them: %v, %v", g.CoverImage, g.Studio)
	}
	// Replacing it still works.
	replaced := owned
	replaced.CoverImage = strPtr("/uploads/mine.jpg")
	push(replaced)
	if g := get(); *g.CoverImage != "/uploads/mine.jpg" {
		t.Fatalf("cover not replaced: %v", *g.CoverImage)
	}

	// Old client inserting a brand-new game gets the played default.
	oldID := uuid.New()
	push(model.VideoGame{ID: oldID, Title: "Old " + oldID.String(), YearsPlayed: []int16{2025}, CreatedAt: time.Now().UTC()})
	if g, err := s.Games.Get(ctx, oldID); err != nil || g.Status != model.GamePlayed {
		t.Fatalf("old-client insert = %+v, %v", g, err)
	}

	// Pull returns the lifecycle fields.
	cs, err := sync.GetChangesSince(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, g := range cs.VideoGames {
		if g.ID == id {
			found = g.Status == model.GameOwned && g.Platform != nil && *g.Platform == "pc"
		}
	}
	if !found {
		t.Fatal("pull did not return the owned game with its platform")
	}
}

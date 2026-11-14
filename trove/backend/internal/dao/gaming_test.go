package dao

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/saifkazi/trove/backend/internal/model"
)

func linkTo(t *testing.T, s *Stores, id uuid.UUID, igdbID int64) {
	t.Helper()
	err := s.Gaming.SaveMeta(context.Background(), model.GameMeta{VideoGameID: id, MatchMethod: model.MatchPicked, IGDBID: &igdbID})
	if err != nil {
		t.Fatal(err)
	}
}

// Relinking replaces a cover and studio the worker filled in from the old
// match, but keeps ones you set.
func TestRelinkClearsOnlyFilledCoverAndStudio(t *testing.T) {
	ctx := context.Background()
	s := testStores(t)

	filled, err := s.Games.Create(ctx, CreateGameParams{Title: "Starfox", Status: model.GameWishlist, YearsPlayed: []int16{}})
	if err != nil {
		t.Fatal(err)
	}
	linkTo(t, s, filled.ID, 1)
	if set, err := s.Gaming.SetCoverIfEmpty(ctx, filled.ID, "/uploads/old.jpg"); err != nil || !set {
		t.Fatalf("set cover: %v %v", set, err)
	}
	if err := s.Gaming.SetStudioIfEmpty(ctx, filled.ID, "Old Publisher"); err != nil {
		t.Fatal(err)
	}

	yours, err := s.Games.Create(ctx, CreateGameParams{
		Title: "Star Fox", Status: model.GameWishlist, YearsPlayed: []int16{},
		CoverImage: strPtr("/uploads/mine.jpg"), Studio: strPtr("My Studio"),
	})
	if err != nil {
		t.Fatal(err)
	}
	linkTo(t, s, yours.ID, 1)

	// Saving the same match again changes nothing.
	linkTo(t, s, filled.ID, 1)
	if g, _ := s.Games.Get(ctx, filled.ID); g.CoverImage == nil {
		t.Fatal("cover cleared without a relink")
	}

	linkTo(t, s, filled.ID, 2)
	linkTo(t, s, yours.ID, 2)
	g, err := s.Games.Get(ctx, filled.ID)
	if err != nil {
		t.Fatal(err)
	}
	if g.CoverImage != nil || g.Studio != nil {
		t.Errorf("filled cover/studio kept after relink: %v %v", g.CoverImage, g.Studio)
	}
	if g.UpdatedAt.Equal(filled.UpdatedAt) {
		t.Error("updated_at not moved, so devices won't see the change")
	}
	mine, err := s.Games.Get(ctx, yours.ID)
	if err != nil {
		t.Fatal(err)
	}
	if mine.CoverImage == nil || *mine.CoverImage != "/uploads/mine.jpg" || mine.Studio == nil || *mine.Studio != "My Studio" {
		t.Errorf("your cover/studio changed: %v %v", mine.CoverImage, mine.Studio)
	}

	// The new match's cover is filled in and can be replaced in turn.
	if set, err := s.Gaming.SetCoverIfEmpty(ctx, filled.ID, "/uploads/new.jpg"); err != nil || !set {
		t.Fatalf("refill cover: %v %v", set, err)
	}
	linkTo(t, s, filled.ID, 3)
	if g, _ := s.Games.Get(ctx, filled.ID); g.CoverImage != nil {
		t.Error("refilled cover kept after another relink")
	}
}

package dao

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/saifkazi/trove/backend/internal/model"
)

func TestPicksStorage(t *testing.T) {
	ctx := context.Background()
	s := testStores(t)
	// Later than any week already stored, so it's the current one even
	// when the test database is reused.
	week := time.Date(2100, 1, 3, 0, 0, 0, 0, time.UTC)
	if last, err := s.Gaming.LastPickRun(ctx); err != nil {
		t.Fatal(err)
	} else if !last.Before(week) {
		week = last.AddDate(0, 0, 7)
	}
	ended := time.Now().Add(-time.Hour)
	open := Pick{Rank: 1, IGDBID: 910001 + week.Unix()%1000, Title: "Open Sale", Pitch: "You'll like it.", Tags: []string{"genre:Puzzle"},
		Store: "steam", StoreRef: "1", URL: "https://example.com", Currency: "INR", BaseMinor: 100000, FinalMinor: 50000, DiscountPct: 50}
	over := open
	over.Rank, over.IGDBID, over.Title, over.SaleEndsAt = 2, open.IGDBID+1, "Sale Over", &ended

	if err := s.Gaming.SavePicks(ctx, week, []Pick{open, over}); err != nil {
		t.Fatal(err)
	}
	if last, err := s.Gaming.LastPickRun(ctx); err != nil || !last.Equal(week) {
		t.Fatalf("last run = %v, %v", last, err)
	}
	current, err := s.Gaming.CurrentPicks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != 1 || current[0].Title != "Open Sale" || current[0].State != PickNew {
		t.Fatalf("current = %+v", current)
	}
	id := current[0].ID

	// Recent picks aren't picked again, even untouched.
	excluded, err := s.Gaming.ExcludedFromPicks(ctx)
	if err != nil || !excluded[open.IGDBID] {
		t.Errorf("excluded = %v, %v", excluded[open.IGDBID], err)
	}

	if err := s.Gaming.ClaimPick(ctx, id, PickDismissed); err != nil {
		t.Fatal(err)
	}
	if err := s.Gaming.ClaimPick(ctx, id, PickWishlisted); !errors.Is(err, ErrPickTaken) {
		t.Errorf("second claim err = %v, want ErrPickTaken", err)
	}
	if current, _ := s.Gaming.CurrentPicks(ctx); len(current) != 0 {
		t.Errorf("dismissed pick still current: %+v", current)
	}
	tags, err := s.Gaming.DismissedPickTags(ctx)
	if err != nil || !slices.ContainsFunc(tags, func(ts []string) bool { return slices.Equal(ts, []string{"genre:Puzzle"}) }) {
		t.Errorf("dismissed tags = %v, %v", tags, err)
	}

	// A failed add puts the pick back.
	if err := s.Gaming.UnclaimPick(ctx, id); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.Gaming.GetPick(ctx, id); p.State != PickNew {
		t.Errorf("state after unclaim = %s", p.State)
	}

	// An empty run is still recorded, so the week isn't retried.
	later := week.AddDate(0, 0, 7)
	if err := s.Gaming.SavePicks(ctx, later, nil); err != nil {
		t.Fatal(err)
	}
	if last, _ := s.Gaming.LastPickRun(ctx); !last.Equal(later) {
		t.Errorf("last run after empty week = %v", last)
	}
}

func TestLibrary(t *testing.T) {
	ctx := context.Background()
	s := testStores(t)
	g, err := s.Games.Create(ctx, CreateGameParams{Title: "Library Game", Status: model.GamePlayed, Rating: ptr16(5), YearsPlayed: []int16{2023, 2025}})
	if err != nil {
		t.Fatal(err)
	}
	igdb := int64(920001)
	if err := s.Gaming.SaveMeta(ctx, model.GameMeta{VideoGameID: g.ID, MatchMethod: model.MatchAuto, IGDBID: &igdb, Genres: []string{"Puzzle"}}); err != nil {
		t.Fatal(err)
	}
	library, err := s.Gaming.Library(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range library {
		if l.IGDBID == igdb {
			if l.LastPlayed != 2025 || *l.Rating != 5 || l.Genres[0] != "Puzzle" {
				t.Errorf("library game = %+v", l)
			}
			return
		}
	}
	t.Error("linked game missing from library")
}

func ptr16(n int16) *int16 { return &n }

package tracker

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/stores"
)

// fakeServices stands in for IGDB, Steam, the eShop and the FX API. steamPrice
// is Elden Ring's current Steam price in paise and can be changed mid-test.
type fakeServices struct {
	url        string
	steamPrice atomic.Int64
}

func newFakeServices(t *testing.T) *fakeServices {
	t.Helper()
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	igdbElden := read("../catalog/testdata/search_elden_ring.json")
	eshopElden := read("../stores/testdata/eshop_search_elden_ring.json")

	f := &fakeServices{}
	f.steamPrice.Store(359900)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			fmt.Fprint(w, `{"access_token":"tok","expires_in":3600}`)
		case r.URL.Path == "/games":
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), `search "elden ring"`) {
				w.Write(igdbElden) //nolint:errcheck
			} else {
				fmt.Fprint(w, `[{"id":1,"name":"Prey","total_rating_count":800},{"id":2,"name":"Prey","total_rating_count":500}]`)
			}
		case r.URL.Path == "/api/appdetails":
			fmt.Fprintf(w, `{"1245620":{"success":true,"data":{"price_overview":{"currency":"INR","initial":359900,"final":%d,"discount_percent":0}}}}`, f.steamPrice.Load())
		case strings.HasPrefix(r.URL.Path, "/1/indexes/"):
			w.Write(eshopElden) //nolint:errcheck
		case r.URL.Path == "/v1/price":
			fmt.Fprint(w, `{"prices":[{"title_id":70010000097552,"sales_status":"onsale","regular_price":{"currency":"USD","raw_value":"79.99"}}]}`)
		case r.URL.Path == "/v1/latest":
			fmt.Fprint(w, `{"rates":{"INR":94.25}}`)
		default:
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	f.url = srv.URL
	return f
}

func newTestTracker(t *testing.T) (*Tracker, *dao.Stores, *fakeServices) {
	t.Helper()
	url := os.Getenv("TROVE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TROVE_TEST_DATABASE_URL not set")
	}
	pool, err := dao.NewPostgresPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	s := dao.NewStores(pool)
	f := newFakeServices(t)
	return &Tracker{
		Store: s.Gaming,
		IGDB:  &catalog.Client{ClientID: "id", ClientSecret: "secret", BaseURL: f.url, TokenURL: f.url + "/token"},
		Steam: &stores.SteamClient{BaseURL: f.url},
		Eshop: &stores.EshopClient{PriceURL: f.url, AlgoliaURL: f.url},
		FX:    &stores.FXClient{BaseURL: f.url},
		Log:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, s, f
}

func addGame(t *testing.T, s *dao.Stores, title, status string, target *int32) uuid.UUID {
	t.Helper()
	g, err := s.Games.Create(context.Background(), dao.CreateGameParams{
		Title: title, Status: status, TargetPriceINR: target, YearsPlayed: []int16{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return g.ID
}

func TestRefreshTracksAWishlistGame(t *testing.T) {
	ctx := context.Background()
	tr, s, fake := newTestTracker(t)
	target := int32(3000)
	id := addGame(t, s, "Elden Ring", model.GameWishlist, &target)

	if err := tr.Refresh(ctx, id); err != nil {
		t.Fatal(err)
	}

	meta, err := s.Gaming.GetMeta(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if meta.MatchMethod != model.MatchAuto || *meta.IGDBID != 119133 || *meta.SteamAppID != "1245620" {
		t.Fatalf("meta = %+v", meta)
	}

	listings, err := s.Gaming.Listings(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, l := range listings {
		got[l.Store] = l.StoreRef
	}
	if got[stores.Steam] != "1245620" || got[stores.Eshop] != "70010000097552" || got[stores.PSN] != "" || len(got) != 3 {
		t.Fatalf("listings = %v", got)
	}

	// Full price, unchanged on a second check: one run, no events.
	if err := tr.CheckPrices(ctx, &id); err != nil {
		t.Fatal(err)
	}
	history, _ := s.Gaming.PriceHistory(ctx, id)
	if len(history) != 2 { // one Steam run + one eShop run
		t.Fatalf("history after two identical checks = %d runs, want 2", len(history))
	}
	if n := pendingEvents(t, s, id); n != 0 {
		t.Fatalf("events at full price = %d", n)
	}

	// Steam drops to ₹2,519 (−30%): below the ₹3,000 target and a new low.
	fake.steamPrice.Store(251900)
	if err := tr.CheckPrices(ctx, &id); err != nil {
		t.Fatal(err)
	}
	if n := pendingEvents(t, s, id); n != 1 {
		t.Fatalf("events after sale = %d, want 1", n)
	}
	// Seeing the same sale again doesn't alert twice.
	if err := tr.CheckPrices(ctx, &id); err != nil {
		t.Fatal(err)
	}
	if n := pendingEvents(t, s, id); n != 1 {
		t.Fatalf("events after repeat check = %d, want 1", n)
	}
	history, _ = s.Gaming.PriceHistory(ctx, id)
	if len(history) != 3 {
		t.Fatalf("history = %d runs, want 3", len(history))
	}
}

func TestAmbiguousGameIsUnresolved(t *testing.T) {
	ctx := context.Background()
	tr, s, _ := newTestTracker(t)
	id := addGame(t, s, "Prey", model.GamePlayed, nil)

	if err := tr.Refresh(ctx, id); err != nil {
		t.Fatal(err)
	}
	meta, err := s.Gaming.GetMeta(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if meta.MatchMethod != model.MatchUnresolved || len(meta.Candidates) != 2 || meta.IGDBID != nil {
		t.Fatalf("meta = %+v", meta)
	}
	// Played games aren't priced.
	if listings, _ := s.Gaming.Listings(ctx, id); len(listings) != 0 {
		t.Fatalf("played game got listings: %v", listings)
	}
}

func pendingEvents(t *testing.T, s *dao.Stores, gameID uuid.UUID) int {
	t.Helper()
	var n int
	err := s.DB.QueryRow(context.Background(), `
		SELECT count(*) FROM sale_events e JOIN store_listings l ON l.id = e.listing_id
		WHERE l.video_game_id = $1 AND e.notified_at IS NULL`, gameID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestFillBlanksOnlyWhenEmpty(t *testing.T) {
	ctx := context.Background()
	tr, s, _ := newTestTracker(t)
	tr.UploadsDir = t.TempDir()
	cover := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "t_cover_big_2x") {
			t.Errorf("cover requested at %s, want the 2x size", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("\xff\xd8\xff fake jpeg")) //nolint:errcheck
	}))
	defer cover.Close()

	link := func(id uuid.UUID) {
		t.Helper()
		g := catalog.Game{
			IGDBID: 1, Title: "x", CoverURL: cover.URL + "/igdb/image/upload/t_cover_big/abc.jpg",
			Publishers: []string{"Team Cherry"}, Developers: []string{"Team Cherry Dev"},
		}
		if err := s.Gaming.SaveMeta(ctx, MetaFromGame(id, g, model.MatchAuto)); err != nil {
			t.Fatal(err)
		}
	}
	bare := addGame(t, s, "No cover "+uuid.NewString(), model.GameWishlist, nil)
	link(bare)
	own, err := s.Games.Create(ctx, dao.CreateGameParams{
		Title: "Own cover " + uuid.NewString(), Status: model.GamePlayed, Studio: func() *string { s := "My Studio"; return &s }(),
		CoverImage: func() *string { p := "/uploads/mine.jpg"; return &p }(), YearsPlayed: []int16{},
	})
	if err != nil {
		t.Fatal(err)
	}
	link(own.ID)

	// Scoped to this test's games: the database may hold others.
	for _, id := range []uuid.UUID{bare, own.ID} {
		if err := tr.FillBlanks(ctx, &id); err != nil {
			t.Fatal(err)
		}
	}

	got, _ := s.Games.Get(ctx, bare)
	if got.CoverImage == nil || !strings.HasPrefix(*got.CoverImage, "/uploads/") {
		t.Fatalf("bare game cover = %v", got.CoverImage)
	}
	if _, err := os.Stat(tr.UploadsDir + "/" + strings.TrimPrefix(*got.CoverImage, "/uploads/")); err != nil {
		t.Fatalf("cover file not saved: %v", err)
	}
	if got.Studio == nil || *got.Studio != "Team Cherry" {
		t.Fatalf("bare game studio = %v, want the publisher", got.Studio)
	}
	mine, _ := s.Games.Get(ctx, own.ID)
	if *mine.CoverImage != "/uploads/mine.jpg" || *mine.Studio != "My Studio" {
		t.Fatalf("your own cover/studio were replaced: %s, %s", *mine.CoverImage, *mine.Studio)
	}
}

func TestStudioFor(t *testing.T) {
	if got := StudioFor([]string{"Bandai Namco"}, []string{"FromSoftware"}); got != "Bandai Namco" {
		t.Errorf("publisher first: got %q", got)
	}
	if got := StudioFor(nil, []string{"Team Cherry"}); got != "Team Cherry" {
		t.Errorf("developer fallback: got %q", got)
	}
	if got := StudioFor(nil, nil); got != "" {
		t.Errorf("nothing: got %q", got)
	}
}

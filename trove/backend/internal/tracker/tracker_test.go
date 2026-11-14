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
	"time"

	"github.com/google/uuid"
	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/push"
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

func TestAutoLinkTakesIGDBTitle(t *testing.T) {
	ctx := context.Background()
	tr, s, _ := newTestTracker(t)
	id := addGame(t, s, "elden ring", model.GamePlayed, nil)
	before, err := s.Games.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	if err := tr.LinkPending(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := s.Games.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if after.Title != "Elden Ring" {
		t.Errorf("title = %q, want IGDB's", after.Title)
	}
	// The rename has to reach other devices.
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Errorf("updated_at not moved: %v -> %v", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestOnlyWishlistedGamesRaiseSaleEvents(t *testing.T) {
	ctx := context.Background()
	tr, s, fake := newTestTracker(t)
	id := addGame(t, s, "Elden Ring", model.GamePlayed, nil)
	if err := tr.Refresh(ctx, id); err != nil {
		t.Fatal(err)
	}
	fake.steamPrice.Store(251900) // −30%
	if err := tr.CheckPrices(ctx, &id); err != nil {
		t.Fatal(err)
	}
	// Priced for its history, but no alert.
	history, _ := s.Gaming.PriceHistory(ctx, id)
	if len(history) != 3 { // two Steam runs + one eShop run
		t.Fatalf("history = %d runs, want 3", len(history))
	}
	if n := pendingEvents(t, s, id); n != 0 {
		t.Errorf("events for a played game = %d, want 0", n)
	}
}

func TestCorrectedTitleIsSearchedAgain(t *testing.T) {
	ctx := context.Background()
	tr, s, _ := newTestTracker(t)
	id := addGame(t, s, "Prey", model.GameWishlist, nil) // ambiguous in the fake IGDB
	if err := tr.Refresh(ctx, id); err != nil {
		t.Fatal(err)
	}
	if meta, _ := s.Gaming.GetMeta(ctx, id); meta.MatchMethod != model.MatchUnresolved {
		t.Fatalf("match = %s, want unresolved", meta.MatchMethod)
	}
	// Refreshing again without a change doesn't search again...
	if err := tr.Refresh(ctx, id); err != nil {
		t.Fatal(err)
	}
	// ...but fixing the title does.
	if err := s.Gaming.SetTitle(ctx, id, "Elden Ring"); err != nil {
		t.Fatal(err)
	}
	if err := tr.Refresh(ctx, id); err != nil {
		t.Fatal(err)
	}
	meta, err := s.Gaming.GetMeta(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if meta.MatchMethod != model.MatchAuto || *meta.IGDBID != 119133 {
		t.Errorf("meta = %+v, want Elden Ring", meta)
	}
}

func TestOnYourDevices(t *testing.T) {
	n64 := catalog.Game{Title: "The Legend of Zelda: Ocarina of Time", ReleaseYear: 1998, Platforms: []int{4}}
	remake := catalog.Game{Title: "The Legend of Zelda: Ocarina of Time", ReleaseYear: 2026, Platforms: []int{catalog.PlatformSwitch2}}
	if got := onYourDevices([]catalog.Game{n64, remake}); len(got) != 1 || got[0].ReleaseYear != 2026 {
		t.Errorf("got %+v, want only the remake", got)
	}
	// Nothing on your devices: keep them all rather than none.
	if got := onYourDevices([]catalog.Game{n64}); len(got) != 1 {
		t.Errorf("got %+v, want the N64 game kept", got)
	}
}

func TestNeedsRelink(t *testing.T) {
	searched := "Donkey Kong Bonanza"
	unresolved := model.GameMeta{MatchMethod: model.MatchUnresolved, Title: &searched}
	if NeedsRelink(unresolved, searched) {
		t.Error("same title searched again")
	}
	if !NeedsRelink(unresolved, "Donkey Kong Bananza") {
		t.Error("corrected title not searched again")
	}
	if !NeedsRelink(model.GameMeta{MatchMethod: model.MatchUnresolved}, "Anything") {
		t.Error("unresolved game from before titles were kept not searched again")
	}
	linked := model.GameMeta{MatchMethod: model.MatchAuto, Title: &searched}
	if NeedsRelink(linked, "Something else") {
		t.Error("a linked game was searched again")
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

type fakeNotifier struct {
	devices int // how many browsers "accept" each notification
	sent    []push.Notification
}

func (f *fakeNotifier) Enabled() bool { return true }
func (f *fakeNotifier) Send(_ context.Context, n push.Notification) (int, error) {
	f.sent = append(f.sent, n)
	return f.devices, nil
}

func TestDeliverSales(t *testing.T) {
	ctx := context.Background()
	tr, s, _ := newTestTracker(t)
	notifier := &fakeNotifier{}
	tr.Notify = notifier

	// sale records a sale event on a fresh Steam listing for the game.
	sale := func(gameID uuid.UUID, ref string) int64 {
		t.Helper()
		if err := s.Gaming.SaveListings(ctx, gameID, []stores.Listing{tr.Steam.ListingFor(ref, "x")}); err != nil {
			t.Fatal(err)
		}
		listings, _ := s.Gaming.Listings(ctx, gameID)
		q := stores.Quote{BaseMinor: 130000, FinalMinor: 91000, DiscountPct: 30}
		if _, err := s.Gaming.AddSaleEvent(ctx, listings[0].ID, "910:open", []string{"sale"}, q); err != nil {
			t.Fatal(err)
		}
		var id int64
		s.DB.QueryRow(ctx, "SELECT id FROM sale_events WHERE listing_id = $1", listings[0].ID).Scan(&id) //nolint:errcheck
		return id
	}
	notified := func(id int64) bool {
		var at *time.Time
		s.DB.QueryRow(ctx, "SELECT notified_at FROM sale_events WHERE id = $1", id).Scan(&at) //nolint:errcheck
		return at != nil
	}

	wanted := addGame(t, s, "Wanted "+uuid.NewString(), model.GameWishlist, nil)
	fresh := sale(wanted, "1")
	bought := addGame(t, s, "Bought "+uuid.NewString(), model.GameOwned, nil)
	boughtSale := sale(bought, "2")
	stale := addGame(t, s, "Stale "+uuid.NewString(), model.GameWishlist, nil)
	staleSale := sale(stale, "3")
	s.DB.Exec(ctx, "UPDATE sale_events SET created_at = now() - interval '4 days' WHERE id = $1", staleSale) //nolint:errcheck

	// No device subscribed: the wanted sale stays pending, the others close.
	if err := tr.DeliverSales(ctx); err != nil {
		t.Fatal(err)
	}
	if notified(fresh) || !notified(boughtSale) || !notified(staleSale) {
		t.Fatalf("after no-device run: fresh=%t bought=%t stale=%t", notified(fresh), notified(boughtSale), notified(staleSale))
	}

	// A phone subscribes: the pending sale is delivered once.
	notifier.devices, notifier.sent = 1, nil
	if err := tr.DeliverSales(ctx); err != nil {
		t.Fatal(err)
	}
	// Other tests share the database and may leave sales pending, so only
	// count notifications for this test's game.
	forWanted := func() int {
		n := 0
		for _, sent := range notifier.sent {
			if strings.HasSuffix(sent.URL, wanted.String()) {
				n++
			}
		}
		return n
	}
	if !notified(fresh) || forWanted() != 1 {
		t.Fatalf("delivery: notified=%t sent=%+v", notified(fresh), notifier.sent)
	}
	if err := tr.DeliverSales(ctx); err != nil || forWanted() != 1 {
		t.Fatalf("sale delivered twice: %v %d", err, forWanted())
	}
}

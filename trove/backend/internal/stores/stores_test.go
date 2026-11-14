package stores

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// serveFixture answers every request with the named testdata file.
func serveFixture(t *testing.T, name string) *httptest.Server {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body) //nolint:errcheck
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestNormalizeTitle(t *testing.T) {
	cases := map[string]string{
		"Mario Kart™ World":                     "mario kart world",
		"Hollow Knight: Silksong":               "hollow knight silksong",
		"Hades II – Nintendo Switch™ 2 Edition": "hades ii",
		"Pokémon Black Version":                 "pokemon black version",
		"Ratchet & Clank":                       "ratchet & clank",
		"Hades II – Nintendo Switch™ 2 Edition": "hades ii",
	}
	for in, want := range cases {
		if got := NormalizeTitle(in); got != want {
			t.Errorf("NormalizeTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMatchScore(t *testing.T) {
	cases := []struct {
		want, got string
		score     int
	}{
		{"elden ring", "elden ring", 2},
		{"elden ring", "elden ring tarnished edition", 1},
		{"stellar blade", "stellar blade complete edition", 1},
		{"celeste", "celeste game of the year edition", 1},
		{"hades", "hades ii", 0},
		{"hades", "hades ii deluxe edition", 0},
		{"hades", "hades uprising", 0},
		{"elden ring", "elden ring nightreign", 0},
	}
	for _, c := range cases {
		if got := MatchScore(c.want, c.got); got != c.score {
			t.Errorf("MatchScore(%q, %q) = %d, want %d", c.want, c.got, got, c.score)
		}
	}
}

func loadHits(t *testing.T, name string) []EshopHit {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var resp struct {
		Results []struct {
			Hits []EshopHit `json:"hits"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	return resp.Results[0].Hits
}

func TestMatchEshop(t *testing.T) {
	cases := []struct {
		fixture, title string
		want           map[string]string // edition -> nsuid
	}{
		{"eshop_search_silksong.json", "Hollow Knight: Silksong",
			map[string]string{"Switch": "70010000020840", "Switch 2": "70010000105851"}},
		{"eshop_search_hades.json", "Hades II",
			map[string]string{"Switch 2": "70010000105526", "Switch": "70010000060675"}},
		{"eshop_search_hades.json", "Hades", map[string]string{"Switch": "70010000033131"}},
		{"eshop_search_elden_ring.json", "Elden Ring", map[string]string{"Switch 2": "70010000097552"}},
		{"eshop_search_elden_ring.json", "Elden Ring Nightreign", map[string]string{}},
	}
	for _, c := range cases {
		got := map[string]string{}
		for _, l := range MatchEshop(c.title, loadHits(t, c.fixture)) {
			if l.Store != Eshop || l.Currency != "USD" || l.URL == "" {
				t.Errorf("%s: malformed listing %+v", c.title, l)
			}
			got[l.Edition] = l.Ref
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: got %v, want %v", c.title, got, c.want)
			continue
		}
		for edition, nsuid := range c.want {
			if got[edition] != nsuid {
				t.Errorf("%s %s: got %q, want %q", c.title, edition, got[edition], nsuid)
			}
		}
	}
}

func TestEshopQuotes(t *testing.T) {
	srv := serveFixture(t, "eshop_price.json")
	c := &EshopClient{PriceURL: srv.URL}
	quotes, err := c.Quotes(context.Background(), []string{"70010000006442", "70010000095431", "70010000000000"})
	if err != nil {
		t.Fatal(err)
	}
	if len(quotes) != 2 {
		t.Fatalf("got %d quotes, want 2 (not_found skipped): %v", len(quotes), quotes)
	}

	celeste := quotes["70010000006442"]
	wantEnd := time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC)
	if celeste.BaseMinor != 1999 || celeste.FinalMinor != 499 || celeste.DiscountPct != 75 || !celeste.OnSale() {
		t.Errorf("celeste = %+v", celeste)
	}
	if celeste.SaleEndsAt == nil || !celeste.SaleEndsAt.Equal(wantEnd) {
		t.Errorf("celeste sale end = %v, want %v", celeste.SaleEndsAt, wantEnd)
	}

	mkw := quotes["70010000095431"]
	if mkw.BaseMinor != 7999 || mkw.FinalMinor != 7999 || mkw.OnSale() || mkw.SaleEndsAt != nil {
		t.Errorf("mario kart world = %+v", mkw)
	}
}

func TestSteamQuotes(t *testing.T) {
	srv := serveFixture(t, "steam_appdetails.json")
	c := &SteamClient{BaseURL: srv.URL}
	quotes, err := c.Quotes(context.Background(), []string{"504230", "1245620", "570", "2000000000"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := quotes["570"]; ok {
		t.Error("free game (data: []) should have no quote")
	}
	if _, ok := quotes["2000000000"]; ok {
		t.Error("unknown app should have no quote")
	}
	elden := quotes["1245620"]
	if elden.Currency != "INR" || elden.BaseMinor != 359900 || elden.Ref != "1245620" {
		t.Errorf("elden ring = %+v", elden)
	}
}

func TestSteamSearch(t *testing.T) {
	srv := serveFixture(t, "steam_search.json")
	listings, err := (&SteamClient{BaseURL: srv.URL}).Search(context.Background(), "celeste")
	if err != nil {
		t.Fatal(err)
	}
	if len(listings) == 0 || listings[0].Ref != "504230" || listings[0].URL != "https://store.steampowered.com/app/504230" {
		t.Errorf("listings = %+v", listings)
	}
}

func TestParseMinor(t *testing.T) {
	cases := map[string]int64{"79.99": 7999, "4.9": 490, "20": 2000, "0.05": 5}
	for in, want := range cases {
		if got, err := parseMinor(in); err != nil || got != want {
			t.Errorf("parseMinor(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := parseMinor("$4.99"); err == nil {
		t.Error("parseMinor should reject currency symbols")
	}
}

func TestPSNListings(t *testing.T) {
	if got := PSNSearchListing("Hollow Knight: Silksong").URL; got != "https://store.playstation.com/en-in/search/Hollow%20Knight:%20Silksong" {
		t.Errorf("search url = %s", got)
	}
	l, ok := PSNProductListing("https://store.playstation.com/en-us/product/UP4133-PPSA04609_00-ELDENRING0000000?foo=1", "Elden Ring")
	if !ok || l.Ref != "UP4133-PPSA04609_00-ELDENRING0000000" ||
		l.URL != "https://store.playstation.com/en-in/product/UP4133-PPSA04609_00-ELDENRING0000000" {
		t.Errorf("product listing = %+v, %t", l, ok)
	}
	for _, bad := range []string{"https://evil.example/en-in/product/X", "https://store.playstation.com/en-in/search/elden", "not a url"} {
		if _, ok := PSNProductListing(bad, "x"); ok {
			t.Errorf("PSNProductListing(%q) accepted", bad)
		}
	}
}

func TestFX(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"amount":1.0,"base":"USD","date":"2026-11-13","rates":{"INR":94.25}}`)) //nolint:errcheck
	}))
	defer srv.Close()
	rate, err := (&FXClient{BaseURL: srv.URL}).USDToINR(context.Background())
	if err != nil || rate != 94.25 {
		t.Errorf("rate = %v, %v", rate, err)
	}
}

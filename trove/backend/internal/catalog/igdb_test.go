package catalog

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeIGDB serves a token endpoint and a games endpoint backed by a fixture.
// The first games call returns 401 when expireFirst is set, to exercise the
// token refresh.
func fakeIGDB(t *testing.T, fixture string, expireFirst bool) (*Client, *atomic.Int32) {
	t.Helper()
	body, err := os.ReadFile("testdata/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	var tokens, calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			n := tokens.Add(1)
			w.Write([]byte(`{"access_token":"tok` + string(rune('0'+n)) + `","expires_in":3600}`)) //nolint:errcheck
		case "/games":
			q, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(q), "version_parent = null") && !strings.Contains(string(q), "where id = (") {
				t.Errorf("unexpected query: %s", q)
			}
			if calls.Add(1) == 1 && expireFirst {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Client-ID") != "id" || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer tok") {
				t.Errorf("missing auth headers: %v", r.Header)
			}
			w.Write(body) //nolint:errcheck
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return &Client{ClientID: "id", ClientSecret: "secret", BaseURL: srv.URL, TokenURL: srv.URL + "/token", RequestGap: time.Nanosecond}, &tokens
}

func TestSearchRanksExactTitleFirst(t *testing.T) {
	c, _ := fakeIGDB(t, "search_elden_ring.json", false)
	games, err := c.Search(context.Background(), "Elden Ring", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 3 {
		t.Fatalf("got %d games, want 3", len(games))
	}
	g := games[0]
	if g.IGDBID != 119133 || g.Title != "Elden Ring" {
		t.Fatalf("top result = %d %q, want 119133 Elden Ring", g.IGDBID, g.Title)
	}
	if g.SteamAppID != "1245620" || g.ReleaseYear != 2022 || g.CoverURL == "" {
		t.Errorf("elden ring fields = %+v", g)
	}
	if !g.HasPlatform(PlatformPC) || !g.HasPlatform(PlatformPS5) || g.HasPlatform(PlatformSwitch) {
		t.Errorf("elden ring platforms = %v", g.Platforms)
	}
	if len(g.Developers) == 0 || g.Developers[0] != "FromSoftware" {
		t.Errorf("developers = %v", g.Developers)
	}
}

func TestQueryRefreshesExpiredToken(t *testing.T) {
	c, tokens := fakeIGDB(t, "search_elden_ring.json", true)
	if _, err := c.Search(context.Background(), "Elden Ring", 1); err != nil {
		t.Fatal(err)
	}
	if n := tokens.Load(); n != 2 {
		t.Errorf("token fetched %d times, want 2 (initial + refresh after 401)", n)
	}
}

func TestMissingCredentials(t *testing.T) {
	_, err := (&Client{}).Search(context.Background(), "x", 1)
	if err == nil || !strings.Contains(err.Error(), "IGDB_CLIENT_ID") {
		t.Errorf("err = %v", err)
	}
}

func TestRank(t *testing.T) {
	games := []Game{
		{Title: "Elden Ring Nightreign", RatingCount: 900},
		{Title: "Elden Ring: Shadow of the Erdtree Edition", RatingCount: 50},
		{Title: "Elden Ring", RatingCount: 3000},
		{Title: "Elden Ring", RatingCount: 2}, // fan remake with the same name
	}
	got := Rank("elden ring", games)
	if got[0].RatingCount != 3000 || got[1].RatingCount != 2 {
		t.Errorf("exact matches not first by popularity: %+v", got)
	}
	if games[0].Title != "Elden Ring Nightreign" {
		t.Error("Rank must not mutate its input")
	}
}

func TestEscape(t *testing.T) {
	if got := escape(`Say "Hi" \o/`); got != `Say \"Hi\" \\o/` {
		t.Errorf("escape = %s", got)
	}
}

func TestEnsureSteamAppID(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Write([]byte(`{"access_token":"tok","expires_in":3600}`)) //nolint:errcheck
			return
		}
		b, _ := io.ReadAll(r.Body)
		query = string(b)
		w.Write([]byte(`[{"id":144024,"name":"Final Fantasy VII Remake Intergrade","external_games":[{"external_game_source":1,"uid":"1462040"}]}]`)) //nolint:errcheck
	}))
	defer srv.Close()
	c := &Client{ClientID: "id", ClientSecret: "s", BaseURL: srv.URL, TokenURL: srv.URL + "/token", RequestGap: time.Nanosecond}

	g, err := c.EnsureSteamAppID(context.Background(), Game{
		IGDBID: 11169, Title: "Final Fantasy VII Remake", Platforms: []int{PlatformPS4, PlatformPC}, BundleIDs: []int64{133299, 144024},
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.SteamAppID != "1462040" {
		t.Errorf("steam app id = %q", g.SteamAppID)
	}
	if !strings.Contains(query, "id = (133299,144024) | version_parent = 11169") {
		t.Errorf("query = %s", query)
	}

	// Already known, or not on PC: no lookup.
	query = ""
	for _, g := range []Game{{SteamAppID: "1", Platforms: []int{PlatformPC}}, {Platforms: []int{PlatformPS5}}} {
		if _, err := c.EnsureSteamAppID(context.Background(), g); err != nil || query != "" {
			t.Errorf("unexpected lookup for %+v: %q", g, query)
		}
	}
}

func TestBySteamAppIDs(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Write([]byte(`{"access_token":"tok","expires_in":3600}`)) //nolint:errcheck
			return
		}
		b, _ := io.ReadAll(r.Body)
		query = string(b)
		w.Write([]byte(`[{"id":3070,"name":"Euro Truck Simulator 2","summary":"Drive across Europe.","external_games":[{"external_game_source":1,"uid":"227300"}]}]`)) //nolint:errcheck
	}))
	defer srv.Close()
	c := &Client{ClientID: "id", ClientSecret: "s", BaseURL: srv.URL, TokenURL: srv.URL + "/token", RequestGap: time.Nanosecond}

	games, err := c.BySteamAppIDs(context.Background(), []string{"227300", "394360"})
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 1 || games[0].SteamAppID != "227300" || games[0].Summary != "Drive across Europe." {
		t.Fatalf("games = %+v", games)
	}
	for _, want := range []string{`external_games.uid = ("227300","394360")`, "external_game_source = 1", "game_type = (0,"} {
		if !strings.Contains(query, want) {
			t.Errorf("query lacks %q: %s", want, query)
		}
	}
}

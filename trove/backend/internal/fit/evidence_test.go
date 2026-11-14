package fit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/saifkazi/trove/backend/internal/catalog"
)

// fixtureServer serves the recorded responses in testdata.
func fixtureServer(t *testing.T, searxUp bool) (*httptest.Server, *int) {
	t.Helper()
	searches := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file := ""
		switch {
		case r.URL.Path == "/api/appdetails":
			file = "steam_appdetails_1245620.json"
		case strings.HasPrefix(r.URL.Path, "/api/v1/reports/summaries/1245620"):
			file = "protondb_1245620.json"
		case r.URL.Path == "/saleaction/ajaxgetdeckappcompatibilityreport":
			file = "deck_1245620.json"
		case r.URL.Path == "/anticheat.json":
			file = "anticheat_games.json"
		case r.URL.Path == "/search" && searxUp:
			searches++
			file = "searx_search.json"
		}
		if file == "" {
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(body) //nolint:errcheck
	}))
	t.Cleanup(srv.Close)
	return srv, &searches
}

func gatherer(srv *httptest.Server) *Gatherer {
	return &Gatherer{SearxURL: srv.URL, SteamURL: srv.URL, ProtonURL: srv.URL, AntiCheatURL: srv.URL + "/anticheat.json"}
}

var eldenRing = Game{
	Title: "Elden Ring", Year: 2022, Genres: []string{"Role-playing (RPG)"},
	Platforms:  []int32{catalog.PlatformPC, catalog.PlatformPS4, catalog.PlatformPS5, catalog.PlatformSwitch2},
	SteamAppID: "1245620",
}

func TestGather(t *testing.T) {
	srv, searches := fixtureServer(t, true)
	ev := gatherer(srv).Gather(context.Background(), eldenRing)

	if !ev.Complete() {
		t.Fatalf("missing %v", ev.Missing)
	}
	if want := []string{PC, PS5, Switch2}; !slices.Equal(ev.Available, want) {
		t.Errorf("available = %v, want %v", ev.Available, want)
	}
	if !ev.Native[PS5] || !ev.Native[Switch2] {
		t.Errorf("native = %v", ev.Native)
	}
	if ev.Steam == nil || ev.Steam.Metacritic != 94 || ev.Steam.Recommendations == 0 {
		t.Fatalf("steam = %+v", ev.Steam)
	}
	if !strings.HasPrefix(ev.Steam.Minimum, "Minimum:; Requires a 64-bit") || strings.Contains(ev.Steam.Minimum, "<") {
		t.Errorf("minimum = %q", ev.Steam.Minimum)
	}
	if ev.Proton == nil || ev.Proton.Tier != "gold" || ev.Proton.Reports != 2107 {
		t.Errorf("proton = %+v", ev.Proton)
	}
	if ev.Deck != "verified" {
		t.Errorf("deck = %q", ev.Deck)
	}
	if ev.AntiCheat == nil || ev.AntiCheat.Status != "Supported" || ev.AntiCheat.BlocksLinux() {
		t.Errorf("anti-cheat = %+v", ev.AntiCheat)
	}
	// One search per device; the same results come back each time and are
	// kept once, capped per query.
	if *searches != 3 || len(ev.Web) != resultsPerQuery {
		t.Errorf("searches = %d, web results = %d", *searches, len(ev.Web))
	}
	if len(ev.Sources) != 3+resultsPerQuery {
		t.Errorf("sources = %v", ev.Sources)
	}
}

func TestGatherMarksMissingSources(t *testing.T) {
	srv, _ := fixtureServer(t, false)
	ev := gatherer(srv).Gather(context.Background(), eldenRing)
	if ev.Complete() || !slices.Equal(ev.Missing, []string{"web search"}) {
		t.Errorf("missing = %v, want only web search", ev.Missing)
	}

	// Not configured counts as missing too, so the verdict is redone later.
	g := gatherer(srv)
	g.SearxURL = ""
	if ev := g.Gather(context.Background(), eldenRing); ev.Complete() {
		t.Error("complete without web search")
	}
}

func TestGatherWithoutSteam(t *testing.T) {
	srv, _ := fixtureServer(t, true)
	game := Game{Title: "Mario Kart World", Platforms: []int32{catalog.PlatformSwitch2}}
	ev := gatherer(srv).Gather(context.Background(), game)
	if !ev.Complete() || ev.Steam != nil || ev.Proton != nil {
		t.Errorf("evidence = %+v", ev)
	}
	if !slices.Equal(ev.Available, []string{Switch2}) {
		t.Errorf("available = %v", ev.Available)
	}
}

func TestAntiCheatBlocksLinux(t *testing.T) {
	srv, _ := fixtureServer(t, true)
	g := gatherer(srv)
	for appid, want := range map[string]bool{"1172470": true, "1085660": true, "1245620": false, "570": false} {
		facts, err := g.antiCheatFor(context.Background(), appid)
		if err != nil {
			t.Fatal(err)
		}
		if facts.BlocksLinux() != want {
			t.Errorf("%s: blocks = %v, want %v", appid, facts.BlocksLinux(), want)
		}
	}
}

func TestPlainText(t *testing.T) {
	in := `<strong>Minimum:</strong><br><ul class="bb_ul"><li><strong>OS:</strong> Windows 10<br></li><li><strong>Memory:</strong> 12 GB RAM &amp; more<br></li></ul>`
	if got, want := plainText(in), "Minimum:; OS: Windows 10; Memory: 12 GB RAM & more"; got != want {
		t.Errorf("plainText = %q, want %q", got, want)
	}
}

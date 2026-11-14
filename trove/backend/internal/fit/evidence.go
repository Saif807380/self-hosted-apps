// Package fit decides which of your devices to play a game on: it gathers
// evidence (where the game is released, PC requirements, how it runs on
// Linux, what reviews say about each version) and asks a model to weigh it.
package fit

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/saifkazi/trove/backend/internal/catalog"
)

// Your devices, as stored in video_games.platform.
const (
	PC      = "pc"
	PS5     = "ps5"
	Switch2 = "switch2"
)

// Devices in display order.
var Devices = []string{PC, PS5, Switch2}

// Game is what's known about a game before gathering evidence (from IGDB).
type Game struct {
	Title      string
	Year       int
	Genres     []string
	Platforms  []int32 // IGDB platform ids
	SteamAppID string  // empty when not on Steam
}

// Evidence is everything a verdict is based on. A nil field means that
// source had nothing, or couldn't be reached (see Complete and Missing).
type Evidence struct {
	Game      Game
	Available []string        // which of your devices the game is released for
	Native    map[string]bool // released natively for PS5 / Switch 2, not only PS4 / Switch
	Steam     *SteamFacts
	Proton    *ProtonFacts
	Deck      string // Steam Deck rating: verified | playable | unsupported | "" (unrated)
	AntiCheat *AntiCheatFacts
	Web       []WebResult
	Sources   []Source
	// Missing names sources that failed; the verdict is redone once they work.
	Missing []string
}

// Complete reports whether every source that applies was reached.
func (e Evidence) Complete() bool { return len(e.Missing) == 0 }

// SteamFacts come from the Steam store page.
type SteamFacts struct {
	Minimum         string // PC requirements as plain text
	Recommended     string
	Metacritic      int
	Recommendations int // Steam's count of positive reviews
}

// ProtonFacts are ProtonDB's summary of how the game runs on Linux.
type ProtonFacts struct {
	Tier       string // platinum | gold | silver | bronze | borked
	Trending   string // tier of recent reports
	Confidence string
	Reports    int
}

// AntiCheatFacts come from AreWeAntiCheatYet, which lists games whose
// anti-cheat may block Linux.
type AntiCheatFacts struct {
	Status     string // Supported | Running | Planned | Broken | Denied
	AntiCheats []string
}

// BlocksLinux is true when the anti-cheat refuses Proton outright.
func (a *AntiCheatFacts) BlocksLinux() bool {
	return a != nil && (a.Status == "Denied" || a.Status == "Broken")
}

// WebResult is one search hit about how a version of the game runs.
type WebResult struct {
	Title   string
	URL     string
	Snippet string
}

// Source is a page the evidence came from, shown under the verdict.
type Source struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

const (
	steamBaseURL    = "https://store.steampowered.com"
	protonBaseURL   = "https://www.protondb.com"
	antiCheatURL    = "https://raw.githubusercontent.com/AreWeAntiCheatYet/AreWeAntiCheatYet/master/games.json"
	antiCheatMaxAge = 24 * time.Hour
	resultsPerQuery = 4
	snippetMax      = 300
)

// Gatherer collects evidence. Zero URLs mean the public services; tests
// point them at fixtures.
type Gatherer struct {
	HTTP         *http.Client
	SearxURL     string // e.g. http://127.0.0.1:8888; empty disables web search
	SteamURL     string
	ProtonURL    string
	AntiCheatURL string

	mu        sync.Mutex
	antiCheat map[string]AntiCheatFacts // by Steam appid
	fetchedAt time.Time
}

// Gather collects evidence for a game. Sources that fail are listed in
// Missing rather than failing the whole gather.
func (g *Gatherer) Gather(ctx context.Context, game Game) Evidence {
	ev := Evidence{Game: game, Available: Available(game.Platforms), Native: native(game.Platforms)}
	missing := func(source string, err error) {
		if err != nil {
			ev.Missing = append(ev.Missing, source)
		}
	}

	if game.SteamAppID != "" && slices.Contains(ev.Available, PC) {
		var err error
		id := game.SteamAppID
		ev.Steam, err = g.steamFacts(ctx, id)
		missing("steam", err)
		ev.Proton, err = g.protonFacts(ctx, id)
		missing("protondb", err)
		ev.Deck, err = g.deckRating(ctx, id)
		missing("steam deck", err)
		ev.AntiCheat, err = g.antiCheatFor(ctx, id)
		missing("anti-cheat list", err)

		ev.Sources = append(ev.Sources,
			Source{"Steam", "https://store.steampowered.com/app/" + id},
			Source{"ProtonDB", "https://www.protondb.com/app/" + id})
		if ev.AntiCheat != nil {
			ev.Sources = append(ev.Sources, Source{"AreWeAntiCheatYet", "https://areweanticheatyet.com"})
		}
	}

	var err error
	ev.Web, err = g.searchWeb(ctx, game.Title, ev.Available, ev.Native)
	missing("web search", err)
	for _, r := range ev.Web {
		ev.Sources = append(ev.Sources, Source{r.Title, r.URL})
	}
	return ev
}

// Available maps IGDB platforms to your devices. Both consoles play their
// predecessor's games, so a PS4 or Switch release counts.
func Available(platforms []int32) []string {
	has := func(ids ...int32) bool {
		return slices.ContainsFunc(platforms, func(p int32) bool { return slices.Contains(ids, p) })
	}
	var out []string
	if has(catalog.PlatformPC) {
		out = append(out, PC)
	}
	if has(catalog.PlatformPS5, catalog.PlatformPS4) {
		out = append(out, PS5)
	}
	if has(catalog.PlatformSwitch2, catalog.PlatformSwitch) {
		out = append(out, Switch2)
	}
	return out
}

func native(platforms []int32) map[string]bool {
	return map[string]bool{
		PS5:     slices.Contains(platforms, catalog.PlatformPS5),
		Switch2: slices.Contains(platforms, catalog.PlatformSwitch2),
	}
}

func (g *Gatherer) steamFacts(ctx context.Context, appid string) (*SteamFacts, error) {
	q := url.Values{"appids": {appid}, "cc": {"in"}, "l": {"english"}}
	var resp map[string]struct {
		Success bool `json:"success"`
		Data    struct {
			// An object, or [] when the page lists no requirements.
			PCRequirements  json.RawMessage     `json:"pc_requirements"`
			Metacritic      struct{ Score int } `json:"metacritic"`
			Recommendations struct{ Total int } `json:"recommendations"`
		} `json:"data"`
	}
	if err := g.getJSON(ctx, or(g.SteamURL, steamBaseURL)+"/api/appdetails?"+q.Encode(), &resp); err != nil {
		return nil, fmt.Errorf("steam appdetails: %w", err)
	}
	app, ok := resp[appid]
	if !ok || !app.Success {
		return nil, nil // delisted or region-locked: nothing to say
	}
	var req struct{ Minimum, Recommended string }
	_ = json.Unmarshal(app.Data.PCRequirements, &req) // [] leaves it empty
	return &SteamFacts{
		Minimum:         plainText(req.Minimum),
		Recommended:     plainText(req.Recommended),
		Metacritic:      app.Data.Metacritic.Score,
		Recommendations: app.Data.Recommendations.Total,
	}, nil
}

func (g *Gatherer) protonFacts(ctx context.Context, appid string) (*ProtonFacts, error) {
	var resp struct {
		Tier         string `json:"tier"`
		TrendingTier string `json:"trendingTier"`
		Confidence   string `json:"confidence"`
		Total        int    `json:"total"`
	}
	err := g.getJSON(ctx, or(g.ProtonURL, protonBaseURL)+"/api/v1/reports/summaries/"+appid+".json", &resp)
	if errorsIsNotFound(err) {
		return nil, nil // no reports yet
	}
	if err != nil {
		return nil, fmt.Errorf("protondb: %w", err)
	}
	return &ProtonFacts{Tier: resp.Tier, Trending: resp.TrendingTier, Confidence: resp.Confidence, Reports: resp.Total}, nil
}

var deckCategories = map[int]string{1: "unsupported", 2: "playable", 3: "verified"}

func (g *Gatherer) deckRating(ctx context.Context, appid string) (string, error) {
	var resp struct {
		Results struct {
			Category int `json:"resolved_category"`
		} `json:"results"`
	}
	u := or(g.SteamURL, steamBaseURL) + "/saleaction/ajaxgetdeckappcompatibilityreport?nAppID=" + appid
	if err := g.getJSON(ctx, u, &resp); err != nil {
		return "", fmt.Errorf("steam deck rating: %w", err)
	}
	return deckCategories[resp.Results.Category], nil
}

// antiCheatFor looks the game up in AreWeAntiCheatYet's list, which is
// downloaded at most daily. Games without anti-cheat aren't listed.
func (g *Gatherer) antiCheatFor(ctx context.Context, appid string) (*AntiCheatFacts, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.antiCheat == nil || time.Since(g.fetchedAt) > antiCheatMaxAge {
		var games []struct {
			Status     string   `json:"status"`
			AntiCheats []string `json:"anticheats"`
			StoreIDs   struct {
				Steam string `json:"steam"`
			} `json:"storeIds"`
		}
		if err := g.getJSON(ctx, or(g.AntiCheatURL, antiCheatURL), &games); err != nil {
			return nil, fmt.Errorf("anti-cheat list: %w", err)
		}
		g.antiCheat = make(map[string]AntiCheatFacts, len(games))
		for _, game := range games {
			if game.StoreIDs.Steam != "" {
				g.antiCheat[game.StoreIDs.Steam] = AntiCheatFacts{Status: game.Status, AntiCheats: game.AntiCheats}
			}
		}
		g.fetchedAt = time.Now()
	}
	if facts, ok := g.antiCheat[appid]; ok {
		return &facts, nil
	}
	return nil, nil
}

// searchWeb asks SearXNG how each available version of the game runs.
func (g *Gatherer) searchWeb(ctx context.Context, title string, available []string, native map[string]bool) ([]WebResult, error) {
	if g.SearxURL == "" {
		return nil, fmt.Errorf("web search not configured")
	}
	var out []WebResult
	seen := map[string]bool{}
	for _, device := range available {
		results, err := g.searx(ctx, searchQuery(title, device, native[device]))
		if err != nil {
			return nil, err
		}
		for _, r := range results {
			if !seen[r.URL] {
				seen[r.URL] = true
				out = append(out, r)
			}
		}
	}
	return out, nil
}

func searchQuery(title, device string, native bool) string {
	switch device {
	case PC:
		return title + " PC port performance"
	case PS5:
		if native {
			return title + " PS5 performance"
		}
		return title + " PS4 game on PS5 performance"
	default:
		if native {
			return title + " Switch 2 performance"
		}
		return title + " Switch 2 backwards compatibility performance"
	}
}

func (g *Gatherer) searx(ctx context.Context, query string) ([]WebResult, error) {
	q := url.Values{"q": {query}, "format": {"json"}}
	var resp struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := g.getJSON(ctx, strings.TrimSuffix(g.SearxURL, "/")+"/search?"+q.Encode(), &resp); err != nil {
		return nil, fmt.Errorf("web search: %w", err)
	}
	out := make([]WebResult, 0, resultsPerQuery)
	for _, r := range resp.Results {
		if len(out) == resultsPerQuery {
			break
		}
		out = append(out, WebResult{Title: r.Title, URL: r.URL, Snippet: truncate(r.Content, snippetMax)})
	}
	return out, nil
}

var (
	breakTags = regexp.MustCompile(`(?i)<br\s*/?>|</li>`)
	anyTag    = regexp.MustCompile(`<[^>]*>`)
	spaces    = regexp.MustCompile(`\s+`)
)

// plainText turns Steam's requirements HTML into one line.
func plainText(s string) string {
	s = breakTags.ReplaceAllString(s, ";")
	s = html.UnescapeString(anyTag.ReplaceAllString(s, " "))
	var parts []string
	for part := range strings.SplitSeq(s, ";") {
		if part = strings.TrimSpace(spaces.ReplaceAllString(part, " ")); part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "; ")
}

func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

func or(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}

// statusError is a non-2xx response.
type statusError struct{ code int }

func (e statusError) Error() string { return "HTTP " + strconv.Itoa(e.code) }

func errorsIsNotFound(err error) bool {
	se, ok := err.(statusError)
	return ok && se.code == http.StatusNotFound
}

var defaultHTTP = &http.Client{Timeout: 20 * time.Second}

func (g *Gatherer) getJSON(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Trove/1.0 (self-hosted)")
	req.Header.Set("Accept", "application/json")
	c := g.HTTP
	if c == nil {
		c = defaultHTTP
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return statusError{resp.StatusCode}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

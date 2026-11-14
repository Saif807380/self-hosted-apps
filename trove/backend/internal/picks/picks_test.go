package picks

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/stores"
)

func stars(n int16) *int16 { return &n }

// library: two loved roguelikes (one similar to Dead Cells), a hated sports
// game, a dropped shooter and a wishlisted platformer.
var library = []LibraryGame{
	{Title: "Hades", Status: "played", Rating: stars(5), LastPlayed: 2026, Genres: []string{"Roguelike", "Action"}, Themes: []string{"Fantasy"}, SimilarIDs: []int64{900}},
	{Title: "Celeste", Status: "played", Rating: stars(5), LastPlayed: 2025, Genres: []string{"Platform", "Indie"}},
	{Title: "FC 25", Status: "played", Rating: stars(1), LastPlayed: 2026, Genres: []string{"Sport"}},
	{Title: "Bad Shooter", Status: "dropped", Genres: []string{"Shooter"}},
	{Title: "Hollow Knight", Status: "wishlist", Genres: []string{"Platform", "Action"}},
	{Title: "Old Favourite", Status: "played", Rating: stars(4), Genres: []string{"Puzzle"}}, // no year: discounted
}

func candidate(id int64, title string, genres []string, rating float64, count int, discount int) Candidate {
	return Candidate{
		Game:    catalog.Game{IGDBID: id, Title: title, Genres: genres, Rating: rating, RatingCount: count},
		Listing: stores.Listing{Store: stores.Steam, Ref: "1", Title: title},
		Quote:   stores.Quote{Currency: "INR", BaseMinor: 100000, FinalMinor: 50000, DiscountPct: discount},
	}
}

func TestBuild(t *testing.T) {
	p := Build(library, nil, 2026)
	if !p.Ready() {
		t.Fatal("profile with four liked games not ready")
	}
	if p.tags["genre:Roguelike"] <= 0 || p.tags["genre:Sport"] >= 0 || p.tags["genre:Shooter"] >= 0 {
		t.Errorf("tags = %v", p.tags)
	}
	// A recent ★5 says more than an old ★5.
	if p.tags["genre:Roguelike"] <= p.tags["genre:Indie"] {
		t.Errorf("recent favourite (%v) not above older one (%v)", p.tags["genre:Roguelike"], p.tags["genre:Indie"])
	}
	if p.Loved[0] != "Hades" || !slices.Contains(p.Disliked, "FC 25") || !slices.Contains(p.Disliked, "Bad Shooter") {
		t.Errorf("loved = %v, disliked = %v", p.Loved, p.Disliked)
	}
	if p.similar[900] != 2 || p.similarTo[900] != "Hades" {
		t.Errorf("similar = %v from %v", p.similar, p.similarTo)
	}

	if Build(library[2:4], nil, 2026).Ready() {
		t.Error("profile with no liked games is ready")
	}
}

func TestDismissedPicksCountAgainstTheirTags(t *testing.T) {
	before := Build(library, nil, 2026)
	after := Build(library, [][]string{{"genre:Roguelike", "keyword:roguelite"}}, 2026)
	if after.tags["genre:Roguelike"] != before.tags["genre:Roguelike"]-1 {
		t.Errorf("genre after dismissal = %v, before %v", after.tags["genre:Roguelike"], before.tags["genre:Roguelike"])
	}
	if after.tags["keyword:roguelite"] != -keywordWeight {
		t.Errorf("keyword after dismissal = %v, want %v", after.tags["keyword:roguelite"], -keywordWeight)
	}
}

func TestShortlistRanksByTaste(t *testing.T) {
	p := Build(library, nil, 2026)
	similar := candidate(900, "Dead Cells", []string{"Platform"}, 85, 200, 50)
	roguelike := candidate(901, "Roguelike B", []string{"Roguelike", "Action"}, 80, 100, 40)
	sports := candidate(902, "FC 26", []string{"Sport"}, 85, 500, 70)
	obscure := candidate(903, "Unknown Platformer", []string{"Platform"}, 95, 2, 50)
	got := Shortlist(p, []Candidate{sports, obscure, roguelike, similar}, 3)
	titles := make([]string, len(got))
	for i, c := range got {
		titles[i] = c.Game.Title
	}
	if !slices.Equal(titles, []string{"Dead Cells", "Roguelike B", "Unknown Platformer"}) {
		t.Errorf("shortlist = %v", titles)
	}
}

type fakeLLM struct {
	reply  string
	err    error
	prompt string
	schema map[string]any
}

func (f *fakeLLM) CompleteJSON(_ context.Context, _, user string, schema map[string]any, out any) (string, error) {
	f.prompt, f.schema = user, schema
	if f.err != nil {
		return "", f.err
	}
	return "claude-sonnet-5-5", json.Unmarshal([]byte(f.reply), out)
}

func TestChoose(t *testing.T) {
	p := Build(library, nil, 2026)
	shortlist := []Candidate{
		candidate(1, "A", []string{"Roguelike"}, 80, 100, 50),
		candidate(2, "B", []string{"Platform"}, 80, 100, 50),
		candidate(3, "C", []string{"Puzzle"}, 80, 100, 50),
	}
	shortlist[1].Game.Summary = "A platformer about climbing."
	// Picks C, repeats it, and names a game not on the shortlist.
	llm := &fakeLLM{reply: `{"picks": [
		{"igdb_id": 3, "pitch": " You'll like C. "},
		{"igdb_id": 3, "pitch": "again"},
		{"igdb_id": 99, "pitch": "made up"}]}`}
	got, err := Choose(context.Background(), llm, p, shortlist, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Game.IGDBID != 3 || got[0].Pitch != "You'll like C." {
		t.Fatalf("picks = %+v", got)
	}
	// The gap is filled from the top of the shortlist with a fallback pitch.
	if got[1].Game.IGDBID != 1 || !strings.Contains(got[1].Pitch, "Roguelike") {
		t.Errorf("filled pick = %+v", got[1])
	}
	for _, want := range []string{"Choose 2 games.", "Games they loved: Hades", "[2] B", "On sale for the PC (Steam).", "A platformer about climbing."} {
		if !strings.Contains(llm.prompt, want) {
			t.Errorf("prompt lacks %q:\n%s", want, llm.prompt)
		}
	}
	enum := llm.schema["properties"].(map[string]any)["picks"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["igdb_id"].(map[string]any)["enum"]
	if !slices.Equal(enum.([]int64), []int64{1, 2, 3}) {
		t.Errorf("id enum = %v", enum)
	}

	want := errors.New("down")
	if _, err := Choose(context.Background(), &fakeLLM{err: want}, p, shortlist, 2); !errors.Is(err, want) {
		t.Errorf("err = %v", err)
	}
}

func TestFallbackPitch(t *testing.T) {
	p := Build(library, nil, 2026)
	if got := FallbackPitch(p, candidate(900, "Dead Cells", nil, 0, 0, 0)); got != "Similar to Hades, which you enjoyed." {
		t.Errorf("similar pitch = %q", got)
	}
	if got := FallbackPitch(p, candidate(5, "X", []string{"Sport", "Roguelike", "Platform"}, 0, 0, 0)); got != "Matches your taste for Platform and Roguelike." { // Celeste ★5 + wishlisted Hollow Knight outweigh Hades ★5
		t.Errorf("genre pitch = %q", got)
	}
	if got := FallbackPitch(p, candidate(6, "Y", []string{"Sport"}, 0, 0, 0)); got != "Well reviewed and on sale." {
		t.Errorf("plain pitch = %q", got)
	}
}

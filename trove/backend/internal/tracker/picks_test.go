package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/stores"
)

func TestPickWeek(t *testing.T) {
	cases := map[string]string{
		"2026-11-15T03:29:00Z": "2026-11-08", // Sunday 08:59 IST: last week's picks
		"2026-11-15T03:30:00Z": "2026-11-15", // Sunday 09:00 IST: this week's
		"2026-11-18T12:00:00Z": "2026-11-15", // Wednesday
		"2026-11-21T20:00:00Z": "2026-11-15", // Saturday 20:00 UTC is Sunday 01:30 IST, before 09:00
	}
	for at, want := range cases {
		now, _ := time.Parse(time.RFC3339, at)
		if got := PickWeek(now).Format(time.DateOnly); got != want {
			t.Errorf("PickWeek(%s) = %s, want %s", at, got, want)
		}
	}
}

// pickStores fakes the Steam specials list and prices, an empty eShop sale
// and IGDB's lookup by Steam appid.
func pickStores(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			fmt.Fprint(w, `{"access_token":"tok","expires_in":3600}`)
		case r.URL.Path == "/search/results/":
			fmt.Fprint(w, `{"items":[
				{"name":"Roguelite Hit","logo":"https://x/steam/apps/111/a/capsule.jpg"},
				{"name":"Sports Thing","logo":"https://x/steam/apps/222/a/capsule.jpg"},
				{"name":"Too Obscure","logo":"https://x/steam/apps/333/a/capsule.jpg"},
				{"name":"Full Price","logo":"https://x/steam/apps/444/a/capsule.jpg"}]}`)
		case r.URL.Path == "/api/appdetails":
			out := map[string]any{}
			for _, id := range strings.Split(r.URL.Query().Get("appids"), ",") {
				final := 50000
				if id == "444" {
					final = 100000
				}
				out[id] = map[string]any{"success": true, "data": map[string]any{"price_overview": map[string]any{
					"currency": "INR", "initial": 100000, "final": final, "discount_percent": (100000 - final) / 1000}}}
			}
			json.NewEncoder(w).Encode(out) //nolint:errcheck
		case r.URL.Path == "/games":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "external_games.uid =") {
				t.Errorf("unexpected IGDB query: %s", body)
			}
			fmt.Fprint(w, `[
				{"id":7001,"name":"Roguelite Hit","genres":[{"name":"Roguelike"}],"total_rating":88,"total_rating_count":300,"external_games":[{"external_game_source":1,"uid":"111"}]},
				{"id":7002,"name":"Sports Thing","genres":[{"name":"Sport"}],"total_rating":80,"total_rating_count":400,"external_games":[{"external_game_source":1,"uid":"222"}]},
				{"id":7003,"name":"Too Obscure","genres":[{"name":"Roguelike"}],"total_rating":95,"total_rating_count":2,"external_games":[{"external_game_source":1,"uid":"333"}]}]`)
		case strings.HasPrefix(r.URL.Path, "/1/indexes/"):
			fmt.Fprint(w, `{"results":[{"hits":[]}]}`)
		default:
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// pickModel chooses only Sports Thing, leaving the rest to the scores.
type pickModel struct{ calls int }

func (m *pickModel) CompleteJSON(_ context.Context, _, _ string, _ map[string]any, out any) (string, error) {
	m.calls++
	return "claude-sonnet-5-5", json.Unmarshal([]byte(`{"picks":[{"igdb_id":7002,"pitch":"A change of pace."}]}`), out)
}

func TestMakePicks(t *testing.T) {
	ctx := context.Background()
	tr, s, _ := newTestTracker(t)
	url := pickStores(t)
	tr.Steam = &stores.SteamClient{BaseURL: url}
	tr.Eshop = &stores.EshopClient{AlgoliaURL: url, PriceURL: url}
	tr.IGDB = &catalog.Client{ClientID: "id", ClientSecret: "s", BaseURL: url, TokenURL: url + "/token", RequestGap: time.Nanosecond}
	chooser := &pickModel{}
	tr.LLM = chooser

	// Three loved roguelikes make a taste profile.
	for i, title := range []string{"Loved One", "Loved Two", "Loved Three"} {
		g, err := s.Games.Create(ctx, dao.CreateGameParams{Title: title, Status: "played", Rating: stars(5), YearsPlayed: []int16{2026}})
		if err != nil {
			t.Fatal(err)
		}
		igdb := int64(8001 + i)
		if err := s.Gaming.SaveMeta(ctx, metaFor(g.ID, igdb, "Roguelike")); err != nil {
			t.Fatal(err)
		}
	}

	week := time.Date(2001, 1, 7, 0, 0, 0, 0, time.UTC) // its own week, apart from other tests
	if err := tr.MakePicks(ctx, week); err != nil {
		t.Fatal(err)
	}
	rows, err := s.Gaming.PicksForWeek(ctx, week)
	if err != nil {
		t.Fatal(err)
	}
	// The model's choice first, then the best score; too few ratings and
	// full price are left out.
	if len(rows) != 2 || rows[0].Title != "Sports Thing" || rows[0].Pitch != "A change of pace." || rows[1].Title != "Roguelite Hit" {
		t.Fatalf("picks = %+v", rows)
	}
	if rows[1].Store != "steam" || rows[1].StoreRef != "111" || rows[1].DiscountPct != 50 || rows[1].Tags[0] != "genre:Roguelike" {
		t.Errorf("pick row = %+v", rows[1])
	}
	if chooser.calls != 1 {
		t.Errorf("model calls = %d", chooser.calls)
	}
}

func metaFor(id uuid.UUID, igdb int64, genre string) model.GameMeta {
	return model.GameMeta{VideoGameID: id, MatchMethod: model.MatchAuto, IGDBID: &igdb, Genres: []string{genre}}
}

func stars(n int16) *int16 { return &n }

package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saifkazi/trove/backend/internal/fit"
	"github.com/saifkazi/trove/backend/internal/llm"
	"github.com/saifkazi/trove/backend/internal/model"
)

// fakeLLM answers every verdict with PC first, or fails with err.
type fakeLLM struct {
	calls int
	err   error
}

func (f *fakeLLM) CompleteJSON(_ context.Context, _, _ string, _ map[string]any, out any) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	reply := `{"summary": "Play it on the G15.", "devices": [
		{"device": "pc", "rating": "great", "expect": "1080p high", "reasons": ["Runs well"]},
		{"device": "ps5", "rating": "good", "expect": "60 fps", "reasons": []}]}`
	return llm.Model, json.Unmarshal([]byte(reply), out)
}

// searx is a SearXNG stand-in that can be switched off. Steam, ProtonDB
// and the anti-cheat list answer 404, which reads as "nothing known".
type searx struct{ up bool }

func (s *searx) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/search" && s.up {
		w.Write([]byte(`{"results": [{"title": "Review", "url": "https://example.com/r", "content": "Runs well"}]}`)) //nolint:errcheck
		return
	}
	if r.URL.Path == "/anticheat.json" {
		w.Write([]byte(`[]`)) //nolint:errcheck
		return
	}
	if r.URL.Path == "/api/appdetails" {
		w.Write([]byte(`{}`)) //nolint:errcheck
		return
	}
	if r.URL.Path == "/saleaction/ajaxgetdeckappcompatibilityreport" {
		w.Write([]byte(`{"results": {}}`)) //nolint:errcheck
		return
	}
	http.NotFound(w, r)
}

func TestJudge(t *testing.T) {
	tr, s, _ := newTestTracker(t)
	ctx := context.Background()
	web := &searx{up: false}
	srv := httptest.NewServer(web)
	t.Cleanup(srv.Close)
	fake := &fakeLLM{}
	tr.Fit = &fit.Gatherer{SearxURL: srv.URL, SteamURL: srv.URL, ProtonURL: srv.URL, AntiCheatURL: srv.URL + "/anticheat.json"}
	tr.LLM = fake

	id := addGame(t, s, "Elden Ring", model.GameWishlist, nil)
	if err := tr.Refresh(ctx, id); err != nil {
		t.Fatal(err)
	}

	// Web search is down: a verdict is made anyway, marked incomplete.
	v, err := s.Gaming.GetVerdict(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if v.Complete || v.Verdict.Devices[0].Device != fit.PC || v.Model != llm.Model {
		t.Errorf("verdict = %+v", v)
	}

	// Still down: the incomplete verdict stands, with no new model call.
	if err := tr.Judge(ctx, &id); err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 {
		t.Errorf("model calls = %d, want 1", fake.calls)
	}

	// Back up: redone and complete. A complete verdict isn't redone.
	web.up = true
	for range 2 {
		if err := tr.Judge(ctx, &id); err != nil {
			t.Fatal(err)
		}
	}
	if v, err = s.Gaming.GetVerdict(ctx, id); err != nil || !v.Complete {
		t.Fatalf("verdict = %+v, err = %v", v, err)
	}
	if fake.calls != 2 {
		t.Errorf("model calls = %d, want 2", fake.calls)
	}
	if len(v.Sources) == 0 || v.Sources[len(v.Sources)-1].URL != "https://example.com/r" {
		t.Errorf("sources = %v", v.Sources)
	}

	// Relinking to another game drops the verdict.
	meta, err := s.Gaming.GetMeta(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	other := int64(999)
	meta.IGDBID = &other
	if err := s.Gaming.SaveMeta(ctx, *meta); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Gaming.GetVerdict(ctx, id); err == nil {
		t.Error("verdict kept after relink")
	}
}

func TestJudgeSavesAPlaceholderOnRefusal(t *testing.T) {
	tr, s, _ := newTestTracker(t)
	ctx := context.Background()
	srv := httptest.NewServer(&searx{up: true})
	t.Cleanup(srv.Close)
	fake := &fakeLLM{err: fmt.Errorf("%w (general_harms)", llm.ErrRefused)}
	tr.Fit = &fit.Gatherer{SearxURL: srv.URL, SteamURL: srv.URL, ProtonURL: srv.URL, AntiCheatURL: srv.URL + "/anticheat.json"}
	tr.LLM = fake

	id := addGame(t, s, "Elden Ring", model.GameWishlist, nil)
	if err := tr.Refresh(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := tr.Judge(ctx, &id); err != nil {
		t.Fatal(err)
	}
	v, err := s.Gaming.GetVerdict(ctx, id)
	if err != nil || !v.Complete || len(v.Verdict.Devices) != 0 {
		t.Errorf("verdict = %+v, err = %v", v, err)
	}
	if fake.calls != 1 {
		t.Errorf("model calls = %d, want 1 (no retry after a refusal)", fake.calls)
	}
}

func TestPlayedGamesGetVerdicts(t *testing.T) {
	tr, s, _ := newTestTracker(t)
	ctx := context.Background()
	srv := httptest.NewServer(&searx{up: true})
	t.Cleanup(srv.Close)
	tr.Fit = &fit.Gatherer{SearxURL: srv.URL, SteamURL: srv.URL, ProtonURL: srv.URL, AntiCheatURL: srv.URL + "/anticheat.json"}
	tr.LLM = &fakeLLM{}

	played := addGame(t, s, "Elden Ring", model.GamePlayed, nil)
	if err := tr.LinkPending(ctx); err != nil {
		t.Fatal(err)
	}
	if err := tr.Judge(ctx, &played); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Gaming.GetVerdict(ctx, played); err != nil {
		t.Errorf("played game has no verdict: %v", err)
	}
}

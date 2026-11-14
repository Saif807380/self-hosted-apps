package fit

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/saifkazi/trove/backend/internal/catalog"
)

// fakeLLM returns a canned reply and records what it was asked.
type fakeLLM struct {
	reply  string
	err    error
	prompt string
	schema map[string]any
	calls  int
}

func (f *fakeLLM) CompleteJSON(_ context.Context, _, user string, schema map[string]any, out any) (string, error) {
	f.calls++
	f.prompt, f.schema = user, schema
	if f.err != nil {
		return "", f.err
	}
	return "claude-sonnet-5-5", json.Unmarshal([]byte(f.reply), out)
}

func evidence(platforms ...int32) Evidence {
	return Evidence{
		Game:      Game{Title: "Test Game", Platforms: platforms, SteamAppID: "1"},
		Available: Available(platforms),
		Native:    native(platforms),
	}
}

func devicesOf(v Verdict) []string {
	out := make([]string, len(v.Devices))
	for i, d := range v.Devices {
		out[i] = d.Device + ":" + d.Rating
	}
	return out
}

func TestDecideRanksAndListsUnavailable(t *testing.T) {
	llm := &fakeLLM{reply: `{"summary": "Play it on PS5.", "devices": [
		{"device": "pc", "rating": "good", "expect": "1080p", "reasons": ["Runs fine"]},
		{"device": "ps5", "rating": "great", "expect": "4K 60", "reasons": ["Best version"]}]}`}
	v, model, err := Decide(context.Background(), llm, evidence(catalog.PlatformPC, catalog.PlatformPS5))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := devicesOf(v), []string{"ps5:great", "pc:good"}; !slices.Equal(got, want) {
		t.Errorf("devices = %v, want %v", got, want)
	}
	if !slices.Equal(v.Unavailable, []string{Switch2}) || model != "claude-sonnet-5-5" {
		t.Errorf("unavailable = %v, model = %q", v.Unavailable, model)
	}
	// The schema only offers released devices.
	enum := llm.schema["properties"].(map[string]any)["devices"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)["device"].(map[string]any)["enum"]
	if !slices.Equal(enum.([]string), []string{PC, PS5}) {
		t.Errorf("device enum = %v", enum)
	}
}

func TestDecideDropsUnreleasedAndDuplicateDevices(t *testing.T) {
	llm := &fakeLLM{reply: `{"summary": "s", "devices": [
		{"device": "switch2", "rating": "great", "expect": "", "reasons": []},
		{"device": "ps5", "rating": "good", "expect": "", "reasons": []},
		{"device": "ps5", "rating": "avoid", "expect": "", "reasons": []}]}`}
	v, _, err := Decide(context.Background(), llm, evidence(catalog.PlatformPS4))
	if err != nil {
		t.Fatal(err)
	}
	if got := devicesOf(v); !slices.Equal(got, []string{"ps5:good"}) {
		t.Errorf("devices = %v", got)
	}
}

func TestDecideAvoidsPCWhenAntiCheatBlocksLinux(t *testing.T) {
	ev := evidence(catalog.PlatformPC, catalog.PlatformPS5)
	ev.AntiCheat = &AntiCheatFacts{Status: "Denied", AntiCheats: []string{"BattlEye"}}
	llm := &fakeLLM{reply: `{"summary": "s", "devices": [
		{"device": "pc", "rating": "great", "expect": "", "reasons": ["Fast"]},
		{"device": "ps5", "rating": "good", "expect": "", "reasons": []}]}`}
	v, _, err := Decide(context.Background(), llm, ev)
	if err != nil {
		t.Fatal(err)
	}
	if got := devicesOf(v); !slices.Equal(got, []string{"ps5:good", "pc:avoid"}) {
		t.Errorf("devices = %v", got)
	}
	if !strings.Contains(v.Devices[1].Reasons[0], "anti-cheat") {
		t.Errorf("reasons = %v", v.Devices[1].Reasons)
	}
	if !strings.Contains(llm.prompt, "Anti-cheat on Linux: Denied (BattlEye)") {
		t.Errorf("prompt lacks anti-cheat:\n%s", llm.prompt)
	}
}

func TestDecideWithNoDevicesSkipsTheModel(t *testing.T) {
	llm := &fakeLLM{}
	v, _, err := Decide(context.Background(), llm, evidence(48000))
	if err != nil || llm.calls != 0 || !slices.Equal(v.Unavailable, Devices) {
		t.Errorf("verdict = %+v, err = %v, calls = %d", v, err, llm.calls)
	}
}

func TestDecidePassesErrorsThrough(t *testing.T) {
	want := errors.New("boom")
	if _, _, err := Decide(context.Background(), &fakeLLM{err: want}, evidence(catalog.PlatformPC)); !errors.Is(err, want) {
		t.Errorf("err = %v", err)
	}
}

func TestPrompt(t *testing.T) {
	ev := evidence(catalog.PlatformPC, catalog.PlatformPS4, catalog.PlatformSwitch)
	ev.Game.Year = 2020
	ev.Steam = &SteamFacts{Minimum: "Minimum:; Memory: 8 GB RAM", Metacritic: 90}
	ev.Deck = "verified"
	ev.Web = []WebResult{{Title: "Review", URL: "https://example.com", Snippet: "Runs well"}}
	ev.Missing = []string{"protondb"}
	got := Prompt(ev)
	for _, want := range []string{
		"Game: Test Game (2020)",
		"Released for: pc, ps5 (PS4 version, played on PS5), switch2 (Switch 1 version, played on Switch 2)",
		"- Minimum:; Memory: 8 GB RAM",
		"- Metacritic (PC): 90",
		"Steam Deck: verified",
		"- Review (https://example.com): Runs well",
		"Couldn't reach: protondb.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt lacks %q:\n%s", want, got)
		}
	}
	// A failed ProtonDB lookup isn't reported as "no reports".
	if strings.Contains(got, "no reports yet") {
		t.Errorf("prompt claims no ProtonDB reports:\n%s", got)
	}
}

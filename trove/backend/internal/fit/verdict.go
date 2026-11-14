package fit

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// Ratings, best first.
const (
	Great    = "great"
	Good     = "good"
	Playable = "playable"
	Avoid    = "avoid"
)

var ratings = []string{Great, Good, Playable, Avoid}

// Verdict ranks the devices a game is released for.
type Verdict struct {
	Summary     string          `json:"summary"`
	Devices     []DeviceVerdict `json:"devices"`     // best first
	Unavailable []string        `json:"unavailable"` // your devices the game isn't released for
}

// DeviceVerdict is how one device would run the game.
type DeviceVerdict struct {
	Device  string   `json:"device"`
	Rating  string   `json:"rating"`
	Expect  string   `json:"expect"` // e.g. "1080p, medium settings, ~60 fps"
	Reasons []string `json:"reasons"`
}

// Completer gets a schema-shaped JSON reply from a model (llm.Client).
type Completer interface {
	CompleteJSON(ctx context.Context, system, user string, schema map[string]any, out any) (model string, err error)
}

// Decide asks the model for a verdict on the evidence, then holds it to what
// the evidence allows: only released devices, and no PC when anti-cheat
// blocks Linux. It returns the model that answered.
func Decide(ctx context.Context, llm Completer, ev Evidence) (Verdict, string, error) {
	if len(ev.Available) == 0 {
		return Verdict{Summary: "Not released for any of your devices.", Unavailable: Devices}, "", nil
	}
	var v Verdict
	model, err := llm.CompleteJSON(ctx, systemPrompt, Prompt(ev), verdictSchema(ev.Available), &v)
	if err != nil {
		return Verdict{}, "", err
	}
	return enforce(v, ev), model, nil
}

func enforce(v Verdict, ev Evidence) Verdict {
	var devices []DeviceVerdict
	seen := map[string]bool{}
	for _, d := range v.Devices {
		if !slices.Contains(ev.Available, d.Device) || seen[d.Device] {
			continue
		}
		seen[d.Device] = true
		if d.Device == PC && ev.AntiCheat.BlocksLinux() && d.Rating != Avoid {
			d.Rating = Avoid
			d.Reasons = append([]string{"Its anti-cheat doesn't work on Linux, so it won't run on the G15"}, d.Reasons...)
		}
		devices = append(devices, d)
	}
	// Best rating first; the model's order breaks ties.
	slices.SortStableFunc(devices, func(a, b DeviceVerdict) int {
		return slices.Index(ratings, a.Rating) - slices.Index(ratings, b.Rating)
	})
	v.Devices = devices
	v.Unavailable = nil
	for _, d := range Devices {
		if !slices.Contains(ev.Available, d) {
			v.Unavailable = append(v.Unavailable, d)
		}
	}
	return v
}

const systemPrompt = `You advise one person on which of their own devices to play a game on. Their devices:

- pc: Dell G15 5520 gaming laptop. Intel Core i7-12700H, NVIDIA RTX 3050 Ti Laptop GPU with only 4 GB of VRAM, 16 GB RAM, 1080p screen. It runs CachyOS Linux, so Windows games run through Proton: ProtonDB and anti-cheat support decide whether a game runs at all. The 4 GB of VRAM is the usual bottleneck in recent games.
- ps5: a standard PlayStation 5. It also plays PS4 games, often with a PS5 upgrade or a boost mode.
- switch2: Nintendo Switch 2, played docked or handheld. It also plays Switch 1 games, some with a free or paid Switch 2 upgrade.

Judge each device the game is released for on how well that version plays: performance, visual quality, port quality and bugs, and what reviewers and players say about that version. Handheld play on the Switch 2 is a plus for slower or shorter-session games. Ignore price; it is shown separately.

Ratings: great (the best way to play it, or as good as any), good (runs well with small compromises), playable (clear compromises such as low resolution, unstable frame rate or missing features), avoid (broken, blocked or much worse than the alternatives).

Base the verdict on the evidence given and on what you reliably know about the game. Where evidence is thin, say so in the reasons rather than inventing frame rates or settings. Keep "expect" short (for example "1080p, medium settings, ~60 fps" or "unknown"). Give at most three reasons per device, each under 15 words, written for the player. The summary is one sentence naming the device to play on and why.`

// Prompt renders the evidence for the model.
func Prompt(ev Evidence) string {
	var b strings.Builder
	g := ev.Game
	fmt.Fprintf(&b, "Game: %s", g.Title)
	if g.Year > 0 {
		fmt.Fprintf(&b, " (%d)", g.Year)
	}
	if len(g.Genres) > 0 {
		fmt.Fprintf(&b, "\nGenres: %s", strings.Join(g.Genres, ", "))
	}
	fmt.Fprintf(&b, "\nReleased for: %s", strings.Join(releasedFor(ev), ", "))

	if s := ev.Steam; s != nil {
		b.WriteString("\n\nSteam:")
		if s.Minimum != "" {
			fmt.Fprintf(&b, "\n- %s", s.Minimum)
		}
		if s.Recommended != "" {
			fmt.Fprintf(&b, "\n- %s", s.Recommended)
		}
		if s.Metacritic > 0 {
			fmt.Fprintf(&b, "\n- Metacritic (PC): %d", s.Metacritic)
		}
		if s.Recommendations > 0 {
			fmt.Fprintf(&b, "\n- Positive Steam reviews: %d", s.Recommendations)
		}
	}
	if p := ev.Proton; p != nil {
		fmt.Fprintf(&b, "\n\nProtonDB (Linux): %s tier, trending %s, %s confidence, %d reports", p.Tier, p.Trending, p.Confidence, p.Reports)
	} else if slices.Contains(ev.Available, PC) && ev.Game.SteamAppID != "" && !slices.Contains(ev.Missing, "protondb") {
		b.WriteString("\n\nProtonDB (Linux): no reports yet")
	}
	if ev.Deck != "" {
		fmt.Fprintf(&b, "\nSteam Deck: %s (a hint for low-power hardware and Linux)", ev.Deck)
	}
	if a := ev.AntiCheat; a != nil {
		fmt.Fprintf(&b, "\nAnti-cheat on Linux: %s (%s)", a.Status, strings.Join(a.AntiCheats, ", "))
	}
	if slices.Contains(ev.Available, PC) && ev.Game.SteamAppID == "" {
		b.WriteString("\n\nNot on Steam: no Steam, ProtonDB or anti-cheat data.")
	}

	if len(ev.Web) > 0 {
		b.WriteString("\n\nWeb search results:")
		for _, r := range ev.Web {
			fmt.Fprintf(&b, "\n- %s (%s): %s", r.Title, r.URL, r.Snippet)
		}
	}
	if len(ev.Missing) > 0 {
		fmt.Fprintf(&b, "\n\nCouldn't reach: %s.", strings.Join(ev.Missing, ", "))
	}
	return b.String()
}

func releasedFor(ev Evidence) []string {
	out := make([]string, 0, len(ev.Available))
	for _, d := range ev.Available {
		switch {
		case d == PS5 && !ev.Native[PS5]:
			out = append(out, "ps5 (PS4 version, played on PS5)")
		case d == Switch2 && !ev.Native[Switch2]:
			out = append(out, "switch2 (Switch 1 version, played on Switch 2)")
		default:
			out = append(out, d)
		}
	}
	return out
}

// verdictSchema limits devices to the ones the game is released for.
func verdictSchema(available []string) map[string]any {
	str := map[string]any{"type": "string"}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"summary", "devices"},
		"properties": map[string]any{
			"summary": str,
			"devices": map[string]any{
				"type":        "array",
				"description": "Every device the game is released for, best first.",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"device", "rating", "expect", "reasons"},
					"properties": map[string]any{
						"device":  map[string]any{"type": "string", "enum": available},
						"rating":  map[string]any{"type": "string", "enum": ratings},
						"expect":  str,
						"reasons": map[string]any{"type": "array", "items": str},
					},
				},
			},
		},
	}
}

package picks

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/saifkazi/trove/backend/internal/stores"
)

// Completer gets a schema-shaped JSON reply from a model (llm.Client).
type Completer interface {
	CompleteJSON(ctx context.Context, system, user string, schema map[string]any, out any) (model string, err error)
}

// Pick is a chosen candidate and why you'd like it.
type Pick struct {
	Candidate
	Pitch string
}

const summaryMax = 400 // characters of each game's IGDB summary in the prompt

// Choose asks the model for the n picks you'd most enjoy from the
// shortlist, each with a pitch. Picks the model leaves out, repeats or makes
// up are made good from the shortlist's order, so it always returns
// min(n, len(shortlist)) picks unless the call fails.
func Choose(ctx context.Context, llm Completer, p Profile, shortlist []Candidate, n int) ([]Pick, error) {
	if len(shortlist) == 0 {
		return nil, nil
	}
	var reply struct {
		Picks []struct {
			IGDBID int64  `json:"igdb_id"`
			Pitch  string `json:"pitch"`
		} `json:"picks"`
	}
	if _, err := llm.CompleteJSON(ctx, systemPrompt, Prompt(p, shortlist, n), schema(shortlist), &reply); err != nil {
		return nil, err
	}
	byID := map[int64]Candidate{}
	for _, c := range shortlist {
		byID[c.Game.IGDBID] = c
	}
	var out []Pick
	chosen := map[int64]bool{}
	for _, r := range reply.Picks {
		c, ok := byID[r.IGDBID]
		if !ok || chosen[r.IGDBID] || len(out) == n {
			continue
		}
		chosen[r.IGDBID] = true
		out = append(out, Pick{Candidate: c, Pitch: strings.TrimSpace(r.Pitch)})
	}
	for _, c := range shortlist {
		if len(out) == n {
			break
		}
		if !chosen[c.Game.IGDBID] {
			chosen[c.Game.IGDBID] = true
			out = append(out, Pick{Candidate: c, Pitch: FallbackPitch(p, c)})
		}
	}
	return out, nil
}

// Fallback picks the n best-scoring candidates without a model.
func Fallback(p Profile, shortlist []Candidate, n int) []Pick {
	out := make([]Pick, 0, n)
	for _, c := range shortlist[:min(n, len(shortlist))] {
		out = append(out, Pick{Candidate: c, Pitch: FallbackPitch(p, c)})
	}
	return out
}

// FallbackPitch says why a candidate was picked without a model.
func FallbackPitch(p Profile, c Candidate) string {
	if title, ok := p.similarTo[c.Game.IGDBID]; ok {
		return fmt.Sprintf("Similar to %s, which you enjoyed.", title)
	}
	var liked []string
	for _, g := range c.Game.Genres {
		if p.tags["genre:"+g] > 0 {
			liked = append(liked, g)
		}
	}
	slices.SortStableFunc(liked, func(a, b string) int { return cmpDesc(p.tags["genre:"+a], p.tags["genre:"+b]) })
	if len(liked) == 0 {
		return "Well reviewed and on sale."
	}
	return "Matches your taste for " + strings.Join(liked[:min(2, len(liked))], " and ") + "."
}

const systemPrompt = `You recommend video games to one person from a shortlist of games that are on sale this week. They play on a Dell G15 gaming laptop (RTX 3050 Ti, Linux), a PS5 and a Nintendo Switch 2.

Choose the games they would most enjoy, judging from the games they loved and disliked. Prefer a varied set over several near-identical games, and prefer well-made games over merely popular ones.

For each pick write a pitch of one or two sentences, under 35 words, addressed to them ("you"). Say what the game is and why it suits them, naming a game they loved when the link is real. Don't mention the price or discount, and don't promise how well it runs on their devices; both are shown separately.`

// Prompt renders your taste and the shortlist for the model.
func Prompt(p Profile, shortlist []Candidate, n int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Choose %d games.\n\nGames they loved: %s", n, joinOrNone(p.Loved))
	fmt.Fprintf(&b, "\nGames they disliked or dropped: %s", joinOrNone(p.Disliked))
	b.WriteString("\n\nShortlist (IGDB id, title, details):")
	for _, c := range shortlist {
		g := c.Game
		fmt.Fprintf(&b, "\n\n[%d] %s", g.IGDBID, g.Title)
		if g.ReleaseYear > 0 {
			fmt.Fprintf(&b, " (%d)", g.ReleaseYear)
		}
		fmt.Fprintf(&b, "\nOn sale for %s.", deviceFor(c.Listing))
		if tags := slices.Concat(g.Genres, g.Themes); len(tags) > 0 {
			fmt.Fprintf(&b, " Genres and themes: %s.", strings.Join(tags, ", "))
		}
		if g.RatingCount > 0 {
			fmt.Fprintf(&b, " Rated %.0f/100 by %d.", g.Rating, g.RatingCount)
		}
		if s := strings.TrimSpace(g.Summary); s != "" {
			fmt.Fprintf(&b, "\n%s", truncate(s, summaryMax))
		}
	}
	return b.String()
}

// deviceFor names the device a deal is for.
func deviceFor(l stores.Listing) string {
	if l.Store == stores.Steam {
		return "the PC (Steam)"
	}
	if l.Edition == "Switch" {
		return "the Switch 2 (Switch version)"
	}
	return "the Switch 2"
}

func schema(shortlist []Candidate) map[string]any {
	ids := make([]int64, len(shortlist))
	for i, c := range shortlist {
		ids[i] = c.Game.IGDBID
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"picks"},
		"properties": map[string]any{
			"picks": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"igdb_id", "pitch"},
					"properties": map[string]any{
						"igdb_id": map[string]any{"type": "integer", "enum": ids},
						"pitch":   map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

func joinOrNone(xs []string) string {
	if len(xs) == 0 {
		return "none recorded"
	}
	return strings.Join(xs, ", ")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}

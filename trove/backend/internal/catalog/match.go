package catalog

import (
	"strings"

	"github.com/saifkazi/trove/backend/internal/stores"
)

// dominantRatio is how many times more ratings the top candidate needs over
// the runner-up to be picked on popularity alone ("Celeste" the 2018 game vs
// a same-named jam game).
const dominantRatio = 10

// LinkPool is how many search results ChooseMatch should see. It needs the
// whole pool: a competitor like "Marvel's Spider-Man 2" can rank below ten
// unrelated games.
const LinkPool = searchPool

// LinkQuery is the IGDB search for a game as you logged it: normalised and
// without an edition name, so "Control Ultimate Edition" searches "control".
func LinkQuery(title string) string {
	return stores.BaseTitle(stores.NormalizeTitle(title))
}

// ChooseMatch decides whether a Trove game (title + optional studio) can be
// linked to an IGDB game without asking. It returns the match and true when
// confident; otherwise the caller should offer the candidates to the user.
//
// Candidates are grouped by how their title relates to the query: exact,
// ending with it ("Marvel's Spider-Man 2" for "Spider-Man 2"), or starting
// with it ("Pokémon Black Version" for "Pokémon Black"). Then:
//  1. One exact title and nothing ending with it: take it. The studio need
//     not match, since Trove's studio is often the publisher.
//  2. Several exact or ending-with titles: keep those whose developer or
//     publisher matches the studio; if one remains, take it. Otherwise take
//     the most rated one if it dominates the rest.
//  3. Only titles that end or start with the query: take one only if it's
//     the single one whose studio matches.
func ChooseMatch(title, studio string, candidates []Game) (Game, bool) {
	want := LinkQuery(title)
	var exact, endsWith, startsWith []Game
	for _, c := range candidates {
		got := stores.NormalizeTitle(c.Title)
		switch {
		case got == want:
			exact = append(exact, c)
		case strings.HasSuffix(got, " "+want):
			endsWith = append(endsWith, c)
		case strings.HasPrefix(got, want+" "):
			startsWith = append(startsWith, c)
		}
	}

	if len(exact) == 1 && len(endsWith) == 0 {
		return exact[0], true
	}
	if len(exact) > 0 {
		contenders := append(exact, endsWith...)
		if byStudio := filterStudio(contenders, studio); len(byStudio) == 1 {
			return byStudio[0], true
		}
		return dominant(contenders)
	}
	if byStudio := filterStudio(append(endsWith, startsWith...), studio); len(byStudio) == 1 {
		return byStudio[0], true
	}
	return Game{}, false
}

func filterStudio(games []Game, studio string) []Game {
	if strings.TrimSpace(studio) == "" {
		return nil
	}
	var out []Game
	for _, g := range games {
		if StudioMatches(studio, g) {
			out = append(out, g)
		}
	}
	return out
}

// StudioMatches compares a free-text studio against a game's developers and
// publishers, ignoring case, spaces and punctuation ("Gamefreak" matches
// "Game Freak", "2K" matches "2K Games").
func StudioMatches(studio string, g Game) bool {
	want := compact(studio)
	if want == "" {
		return false
	}
	for _, company := range append(append([]string{}, g.Developers...), g.Publishers...) {
		got := compact(company)
		if got == want {
			return true
		}
		// Containment only for names long enough not to match by accident
		// ("ea" is inside "steamworks").
		if len(want) >= 3 && len(got) >= 3 && (strings.Contains(got, want) || strings.Contains(want, got)) {
			return true
		}
	}
	return false
}

func compact(s string) string {
	return strings.ReplaceAll(stores.NormalizeTitle(s), " ", "")
}

func dominant(games []Game) (Game, bool) {
	ranked := Rank("", games) // only popularity differs among same-title games
	if len(ranked) > 1 && ranked[0].RatingCount < dominantRatio*max(ranked[1].RatingCount, 1) {
		return Game{}, false
	}
	return ranked[0], true
}

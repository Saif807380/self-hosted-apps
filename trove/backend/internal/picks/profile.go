// Package picks recommends games on sale that you'd probably like: a taste
// profile from your library, a score for each candidate, and a model that
// chooses the week's picks from a shortlist and says why.
package picks

import (
	"math"
	"slices"
	"strings"

	"github.com/saifkazi/trove/backend/internal/catalog"
)

// LibraryGame is a game in your Trove library, as the taste profile sees it.
type LibraryGame struct {
	Title      string
	Status     string // wishlist | owned | played | dropped
	Rating     *int16 // 1–5 stars, played games
	LastPlayed int    // latest year played; 0 when unknown
	IGDBID     int64
	Genres     []string
	Themes     []string
	Keywords   []string
	SimilarIDs []int64
}

// Tag weights by kind: keywords are many and noisy, genres and themes few.
const (
	genreWeight   = 1.0
	themeWeight   = 1.0
	keywordWeight = 0.4
	// recencyDecay shrinks a played game's say for each year since you
	// last played it; a game with no year played counts as a few years old.
	recencyDecay  = 0.85
	unknownYearAt = 3
	// dismissedWeight is how much a pick you turned down counts against
	// its tags.
	dismissedWeight = -1.0
	// minLiked is how many games you must like before picks make sense.
	minLiked = 3
)

// Profile is your taste: weights for tags (genres, themes, keywords) and for
// IGDB games similar to the ones you liked.
type Profile struct {
	tags    map[string]float64
	similar map[int64]float64
	// similarTo names the liked game each similar game comes from, for a
	// fallback pitch ("Similar to Hades, which you loved").
	similarTo map[int64]string
	// Loved and Disliked are titles for the model, best and worst first.
	Loved    []string
	Disliked []string
	liked    int
}

// Ready reports whether there's enough to go on.
func (p Profile) Ready() bool { return p.liked >= minLiked }

// gameWeight is how much a game says about your taste: positive for games
// you rated well, want or own, negative for low ratings and dropped games.
func gameWeight(g LibraryGame, thisYear int) float64 {
	switch g.Status {
	case "wishlist", "owned":
		return 1
	case "dropped":
		return -1.5
	}
	w := 0.5 // played, unrated
	if g.Rating != nil && *g.Rating > 0 {
		w = float64(*g.Rating) - 3 // ★5 = +2 … ★1 = −2
	}
	age := unknownYearAt
	if g.LastPlayed > 0 {
		age = max(0, thisYear-g.LastPlayed)
	}
	return w * math.Pow(recencyDecay, float64(age))
}

// Build makes a profile from your library and the tags of picks you turned
// down.
func Build(library []LibraryGame, dismissedTags [][]string, thisYear int) Profile {
	p := Profile{tags: map[string]float64{}, similar: map[int64]float64{}, similarTo: map[int64]string{}}
	type weighted struct {
		title string
		w     float64
	}
	var ranked []weighted
	for _, g := range library {
		w := gameWeight(g, thisYear)
		if w == 0 {
			continue
		}
		for tag, tw := range tagWeights(g.Genres, g.Themes, g.Keywords) {
			p.tags[tag] += w * tw
		}
		if w > 0 {
			p.liked++
			for _, id := range g.SimilarIDs {
				if w > p.similar[id] {
					p.similarTo[id] = g.Title
				}
				p.similar[id] = max(p.similar[id], w)
			}
		}
		ranked = append(ranked, weighted{g.Title, w})
	}
	for _, tags := range dismissedTags {
		for _, tag := range tags {
			p.tags[tag] += dismissedWeight * kindWeight(tag)
		}
	}
	slices.SortStableFunc(ranked, func(a, b weighted) int { return cmpDesc(a.w, b.w) })
	for _, r := range ranked {
		if r.w >= 1 && len(p.Loved) < 12 {
			p.Loved = append(p.Loved, r.title)
		}
	}
	for i := len(ranked) - 1; i >= 0 && len(p.Disliked) < 6; i-- {
		if ranked[i].w < 0 {
			p.Disliked = append(p.Disliked, ranked[i].title)
		}
	}
	return p
}

// Tags returns a game's tags as stored for a pick, so a dismissed pick can
// count against them later.
func Tags(g catalog.Game) []string {
	out := make([]string, 0, len(g.Genres)+len(g.Themes)+len(g.Keywords))
	for tag := range tagWeights(g.Genres, g.Themes, g.Keywords) {
		out = append(out, tag)
	}
	slices.Sort(out)
	return out
}

// tagWeights namespaces tags by kind ("genre:Platform") with their weights.
func tagWeights(genres, themes, keywords []string) map[string]float64 {
	out := map[string]float64{}
	for _, g := range genres {
		out["genre:"+g] = genreWeight
	}
	for _, t := range themes {
		out["theme:"+t] = themeWeight
	}
	for _, k := range keywords {
		out["keyword:"+k] = keywordWeight
	}
	return out
}

// kindWeight is the weight of a namespaced tag's kind.
func kindWeight(tag string) float64 {
	switch {
	case strings.HasPrefix(tag, "keyword:"):
		return keywordWeight
	case strings.HasPrefix(tag, "theme:"):
		return themeWeight
	default:
		return genreWeight
	}
}

func cmpDesc(a, b float64) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	}
	return 0
}

package picks

import (
	"math"
	"slices"

	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/stores"
)

// Candidate is a game on sale: its IGDB entry and its discounted listing.
type Candidate struct {
	Game    catalog.Game
	Listing stores.Listing
	Quote   stores.Quote
}

const (
	similarWeight  = 2.0 // IGDB lists it as similar to a game you liked
	qualityWeight  = 2.0 // well reviewed by many
	discountWeight = 0.5 // a deeper discount tips close calls
	// ratingsForFullTrust is how many ratings make IGDB's score count fully.
	ratingsForFullTrust = 50
)

// Score is how likely you are to enjoy the candidate: shared tags with your
// taste, similarity to games you liked, review quality and the discount.
func Score(p Profile, c Candidate) float64 {
	tags := tagWeights(c.Game.Genres, c.Game.Themes, c.Game.Keywords)
	var tagScore float64
	for tag, w := range tags {
		tagScore += p.tags[tag] * w
	}
	if len(tags) > 0 {
		tagScore /= math.Sqrt(float64(len(tags))) // many tags shouldn't win by number
	}
	trust := math.Min(1, float64(c.Game.RatingCount)/ratingsForFullTrust)
	quality := c.Game.Rating / 100 * trust
	return tagScore + similarWeight*p.similar[c.Game.IGDBID] + qualityWeight*quality +
		discountWeight*float64(c.Quote.DiscountPct)/100
}

// Shortlist returns the n best-scoring candidates, best first.
func Shortlist(p Profile, candidates []Candidate, n int) []Candidate {
	type scored struct {
		c     Candidate
		score float64
	}
	all := make([]scored, len(candidates))
	for i, c := range candidates {
		all[i] = scored{c, Score(p, c)}
	}
	slices.SortStableFunc(all, func(a, b scored) int { return cmpDesc(a.score, b.score) })
	out := make([]Candidate, 0, n)
	for _, s := range all[:min(n, len(all))] {
		out = append(out, s.c)
	}
	return out
}

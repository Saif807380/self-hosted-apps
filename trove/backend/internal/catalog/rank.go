package catalog

import (
	"cmp"
	"slices"
	"strings"

	"github.com/saifkazi/trove/backend/internal/stores"
)

// Rank orders search results so the game the user most likely meant comes
// first: an exact title match beats a title that starts or ends with the
// query, which beats everything else; ties go to the more-rated game.
func Rank(query string, games []Game) []Game {
	want := stores.NormalizeTitle(query)
	ranked := slices.Clone(games)
	slices.SortStableFunc(ranked, func(a, b Game) int {
		return cmp.Or(
			cmp.Compare(titleScore(want, b.Title), titleScore(want, a.Title)),
			cmp.Compare(b.RatingCount, a.RatingCount),
		)
	})
	return ranked
}

func titleScore(want, title string) int {
	got := stores.NormalizeTitle(title)
	switch {
	case got == want:
		return 2
	case strings.HasPrefix(got, want+" "), strings.HasSuffix(got, " "+want):
		return 1
	default:
		return 0
	}
}

package stores

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// editionSuffixes are stripped before comparing titles, so a store's
// "X – Nintendo Switch 2 Edition" still matches IGDB's "X".
var editionSuffixes = []string{
	"nintendo switch 2 edition",
	"switch 2 edition",
	"ps5",
	"ps4 & ps5",
	"ps4",
}

// stripAccents turns "Pokémon" into "Pokemon".
var stripAccents = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// NormalizeTitle lowercases a title and drops accents, trademark marks,
// punctuation and edition suffixes so titles from different catalogs can be
// compared.
func NormalizeTitle(s string) string {
	if plain, _, err := transform.String(stripAccents, s); err == nil {
		s = plain
	}
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r == '™' || r == '®' || r == '©':
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '&':
			b.WriteString(" & ")
		default:
			b.WriteRune(' ')
		}
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	for _, suffix := range editionSuffixes {
		if trimmed, ok := strings.CutSuffix(out, " "+suffix); ok {
			return trimmed
		}
	}
	return out
}

// maxEditionWords bounds how much a "<name> Edition" suffix may add
// ("game of the year edition" is the longest common one).
const maxEditionWords = 5

// sequelMarkers start a sequel's title, so "hades ii deluxe edition" is never
// treated as an edition of "hades".
var sequelMarkers = map[string]bool{
	"2": true, "3": true, "4": true, "5": true, "6": true, "7": true, "8": true, "9": true,
	"ii": true, "iii": true, "iv": true, "v": true, "vi": true, "vii": true, "viii": true, "ix": true, "x": true,
}

// MatchScore compares two normalised titles: 2 for identical, 1 when got is a
// named edition of want ("elden ring tarnished edition"), 0 otherwise.
func MatchScore(want, got string) int {
	if got == want {
		return 2
	}
	extra, ok := strings.CutPrefix(got, want+" ")
	if !ok {
		return 0
	}
	words := strings.Fields(extra)
	if words[len(words)-1] != "edition" || len(words) > maxEditionWords || sequelMarkers[words[0]] {
		return 0
	}
	return 1
}

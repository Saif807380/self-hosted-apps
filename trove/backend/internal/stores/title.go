package stores

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// platformSuffixes are stripped before comparing titles, so a store's
// "X – Nintendo Switch 2 Edition" still matches IGDB's "X". Longest first.
var platformSuffixes = []string{
	"nintendo switch 2 edition",
	"switch 2 edition",
	"ps4 and ps5",
	"ps5",
	"ps4",
}

// romanNumerals become digits so "Witcher III" matches "Witcher 3". "i" is
// left alone: it's far more often a word than a numeral.
var romanNumerals = map[string]string{
	"ii": "2", "iii": "3", "iv": "4", "v": "5", "vi": "6", "vii": "7", "viii": "8", "ix": "9", "x": "10",
}

// stripAccents turns "Pokémon" into "Pokemon".
var stripAccents = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// NormalizeTitle reduces a title to a comparable form: lowercase, no accents,
// trademark marks, apostrophes or punctuation, "&" spelled "and", Roman
// numerals as digits, and platform edition suffixes removed.
func NormalizeTitle(s string) string {
	if plain, _, err := transform.String(stripAccents, s); err == nil {
		s = plain
	}
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r == '™' || r == '®' || r == '©' || r == '\'' || r == '’':
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '&':
			b.WriteString(" and ")
		default:
			b.WriteRune(' ')
		}
	}
	words := strings.Fields(b.String())
	for i, w := range words {
		if digit, ok := romanNumerals[w]; ok {
			words[i] = digit
		}
	}
	out := strings.Join(words, " ")
	for _, suffix := range platformSuffixes {
		if trimmed, ok := strings.CutSuffix(out, " "+suffix); ok {
			return trimmed
		}
	}
	return out
}

// maxEditionWords bounds how much a "<name> Edition" suffix may add
// ("game of the year edition" is the longest common one).
const maxEditionWords = 5

// sequelMarkers start a sequel's title, so "hades 2 deluxe edition" is never
// treated as an edition of "hades". Roman numerals are already digits.
var sequelMarkers = map[string]bool{
	"2": true, "3": true, "4": true, "5": true, "6": true, "7": true, "8": true, "9": true, "10": true,
}

// MatchScore compares two normalised titles: 2 for identical, 1 when got is a
// named edition of want ("elden ring tarnished edition"), 0 otherwise.
func MatchScore(want, got string) int {
	if got == want {
		return 2
	}
	extra, ok := strings.CutPrefix(got, want+" ")
	if !ok || !isEditionSuffix(extra) {
		return 0
	}
	return 1
}

// editionNames are the edition labels people log games under. BaseTitle
// strips only these; a general "<anything> edition" rule would turn "The Last
// of Us Remastered Edition" into "the". Longest first.
var editionNames = []string{
	"game of the year edition", "digital deluxe edition", "directors cut",
	"goty edition", "ultimate edition", "complete edition", "deluxe edition",
	"definitive edition", "standard edition", "special edition", "premium edition",
	"enhanced edition", "legendary edition", "anniversary edition", "collectors edition",
	"gold edition", "goty",
}

// BaseTitle strips a trailing edition name from a normalised title, so a game
// logged as "Control Ultimate Edition" or "Arkham City GOTY" is searched and
// matched as the game itself.
func BaseTitle(normalized string) string {
	for _, name := range editionNames {
		if base, ok := strings.CutSuffix(normalized, " "+name); ok {
			return base
		}
	}
	return normalized
}

func isEditionSuffix(extra string) bool {
	words := strings.Fields(extra)
	return len(words) > 0 && len(words) <= maxEditionWords &&
		words[len(words)-1] == "edition" && !sequelMarkers[words[0]]
}

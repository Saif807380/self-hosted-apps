package catalog

import "testing"

func TestChooseMatch(t *testing.T) {
	game := func(id int64, title string, ratings int, companies ...string) Game {
		return Game{IGDBID: id, Title: title, RatingCount: ratings, Developers: companies}
	}
	cases := []struct {
		name, title, studio string
		candidates          []Game
		wantID              int64 // 0 = not confident
	}{
		{"unique exact title, studio is the publisher", "Star Wars Jedi: Fallen Order", "EA",
			[]Game{game(1, "Star Wars Jedi: Fallen Order", 900, "Respawn"), game(2, "Star Wars Jedi: Survivor", 500, "Respawn")}, 1},
		{"same title, studio picks", "Celeste", "Extremely OK Games",
			[]Game{game(1, "Celeste", 5, "Jam Team"), game(2, "Celeste", 3000, "Extremely OK Games")}, 2},
		{"same title, no studio, one dominates", "Celeste", "",
			[]Game{game(1, "Celeste", 5), game(2, "Celeste", 3000)}, 2},
		{"same title, no studio, close popularity", "Prey", "",
			[]Game{game(1, "Prey", 800), game(2, "Prey", 400)}, 0},
		{"prefix with matching studio", "Pokemon Black", "Gamefreak",
			[]Game{game(1, "Pokémon Black Version", 700, "Game Freak"), game(2, "Pokémon White Version", 650, "Game Freak")}, 1},
		{"two prefixes with matching studio", "Pokemon Black", "Gamefreak",
			[]Game{game(1, "Pokémon Black Version", 700, "Game Freak"), game(2, "Pokémon Black Version 2", 500, "Game Freak")}, 0},
		{"prefix without studio", "Pokemon Black", "",
			[]Game{game(1, "Pokémon Black Version", 700, "Game Freak")}, 0},
		{"exact title competes with a branded one", "Spider-man 2", "",
			[]Game{game(1, "Spider-Man 2", 300, "Treyarch"), game(2, "Marvel's Spider-Man 2", 900, "Insomniac Games")}, 0},
		{"branded one wins on studio", "Spider-man 2", "Insomniac",
			[]Game{game(1, "Spider-Man 2", 300, "Treyarch"), game(2, "Marvel's Spider-Man 2", 900, "Insomniac Games")}, 2},
		{"suffix only, no studio match", "FC 24", "EA",
			[]Game{game(1, "EA Sports FC 24", 400, "EA Vancouver")}, 0},
		{"edition in the logged title", "Control Ultimate Edition", "Remedy",
			[]Game{game(1, "Control", 2000, "Remedy Entertainment"), game(2, "Control: AWE", 100, "Remedy Entertainment")}, 1},
		{"roman numerals and apostrophes", "Assassins Creed II", "Ubisoft",
			[]Game{game(1, "Assassin's Creed II", 2000, "Ubisoft Montreal"), game(2, "Assassin's Creed III", 1800, "Ubisoft Montreal")}, 1},
		{"nothing close", "NBA 2K27", "2K",
			[]Game{game(1, "NBA 2K26", 50, "Visual Concepts")}, 0},
	}
	for _, c := range cases {
		got, ok := ChooseMatch(c.title, c.studio, c.candidates)
		if (c.wantID == 0) != !ok || (ok && got.IGDBID != c.wantID) {
			t.Errorf("%s: got %d (ok=%t), want %d", c.name, got.IGDBID, ok, c.wantID)
		}
	}
}

func TestStudioMatches(t *testing.T) {
	g := Game{Developers: []string{"Game Freak"}, Publishers: []string{"2K Games", "Steamworks"}}
	for studio, want := range map[string]bool{
		"Gamefreak": true, "game freak": true, "2K": false, "2K Games": true, "EA": false, "": false,
	} {
		if got := StudioMatches(studio, g); got != want {
			t.Errorf("StudioMatches(%q) = %t, want %t", studio, got, want)
		}
	}
}

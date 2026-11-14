// Package catalog looks games up in IGDB, the canonical identity Trove uses to
// tie a game to its store listings and to the tags recommendations work from.
package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	igdbURL  = "https://api.igdb.com/v4"
	tokenURL = "https://id.twitch.tv/oauth2/token"

	sourceSteam = 1 // external_game_source for Steam

	coverURL = "https://images.igdb.com/igdb/image/upload/t_cover_big/%s.jpg"
)

// IGDB platform ids for the devices Trove cares about.
const (
	PlatformPC      = 6
	PlatformPS4     = 48
	PlatformPS5     = 167
	PlatformSwitch  = 130
	PlatformSwitch2 = 508
)

// mainGameTypes are IGDB game_type values that are games in their own right:
// main game, standalone expansion, remake, remaster, expanded game.
const mainGameTypes = "(0,4,8,9,10)"

const gameFields = `fields name, first_release_date, cover.image_id, platforms,
 involved_companies.company.name, involved_companies.developer,
 genres.name, themes.name, keywords.name, total_rating, total_rating_count, similar_games,
 external_games.external_game_source, external_games.uid;`

// Game is the subset of an IGDB game Trove uses.
type Game struct {
	IGDBID      int64
	Title       string
	ReleaseYear int
	CoverURL    string
	Platforms   []int
	Developers  []string
	Genres      []string
	Themes      []string
	Keywords    []string
	Rating      float64 // 0–100, critic + user aggregate; 0 when unrated
	RatingCount int
	SimilarIDs  []int64
	SteamAppID  string
}

// HasPlatform reports whether the game was released on any of the platforms.
func (g Game) HasPlatform(ids ...int) bool {
	for _, p := range g.Platforms {
		for _, id := range ids {
			if p == id {
				return true
			}
		}
	}
	return false
}

// Client talks to IGDB using Twitch client credentials.
type Client struct {
	ClientID     string
	ClientSecret string
	HTTP         *http.Client
	BaseURL      string // overridden in tests
	TokenURL     string // overridden in tests

	mu      sync.Mutex
	token   string
	expires time.Time
}

// searchPool is how many IGDB results Search ranks before truncating. IGDB's
// own ordering often puts a spin-off or fan game above the exact title.
const searchPool = 25

// Search finds games by name, best match first (see Rank).
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Game, error) {
	body := fmt.Sprintf(`search "%s"; %s where game_type = %s & version_parent = null; limit %d;`,
		escape(query), gameFields, mainGameTypes, searchPool)
	games, err := c.games(ctx, body)
	if err != nil {
		return nil, err
	}
	games = Rank(query, games)
	if len(games) > limit {
		games = games[:limit]
	}
	return games, nil
}

// Get fetches games by IGDB id.
func (c *Client) Get(ctx context.Context, ids []int64) ([]Game, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	body := fmt.Sprintf(`%s where id = (%s); limit %d;`, gameFields, strings.Join(parts, ","), len(ids))
	return c.games(ctx, body)
}

func (c *Client) games(ctx context.Context, body string) ([]Game, error) {
	var raw []igdbGame
	if err := c.query(ctx, "games", body, &raw); err != nil {
		return nil, err
	}
	out := make([]Game, len(raw))
	for i, g := range raw {
		out[i] = g.toGame()
	}
	return out, nil
}

// query posts an Apicalypse body, refreshing the token once on a 401.
func (c *Client) query(ctx context.Context, endpoint, body string, out any) error {
	for attempt := 0; ; attempt++ {
		token, err := c.accessToken(ctx)
		if err != nil {
			return err
		}
		err = c.post(ctx, endpoint, body, token, out)
		if errors.Is(err, errUnauthorized) && attempt == 0 {
			c.mu.Lock()
			c.token = ""
			c.mu.Unlock()
			continue
		}
		return err
	}
}

var errUnauthorized = errors.New("igdb: unauthorized")

func (c *Client) post(ctx context.Context, endpoint, body, token string, out any) error {
	base := igdbURL
	if c.BaseURL != "" {
		base = c.BaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/"+endpoint, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Client-ID", c.ClientID)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http().Do(req)
	if err != nil {
		return fmt.Errorf("igdb %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return errUnauthorized
	}
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("igdb %s: HTTP %d: %s", endpoint, resp.StatusCode, msg)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		return "", errors.New("igdb: IGDB_CLIENT_ID and IGDB_CLIENT_SECRET must be set")
	}
	endpoint := tokenURL
	if c.TokenURL != "" {
		endpoint = c.TokenURL
	}
	form := url.Values{
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"grant_type":    {"client_credentials"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http().Do(req)
	if err != nil {
		return "", fmt.Errorf("igdb token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("igdb token: HTTP %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("igdb token: %w", err)
	}
	c.token = tok.AccessToken
	// Refresh a minute early so a request never races the expiry.
	c.expires = time.Now().Add(time.Duration(tok.ExpiresIn)*time.Second - time.Minute)
	return c.token, nil
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func escape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}

type named struct {
	Name string `json:"name"`
}

type igdbGame struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Released int64  `json:"first_release_date"`
	Cover    *struct {
		ImageID string `json:"image_id"`
	} `json:"cover"`
	Platforms []int `json:"platforms"`
	Companies []struct {
		Company   named `json:"company"`
		Developer bool  `json:"developer"`
	} `json:"involved_companies"`
	Genres      []named `json:"genres"`
	Themes      []named `json:"themes"`
	Keywords    []named `json:"keywords"`
	Rating      float64 `json:"total_rating"`
	RatingCount int     `json:"total_rating_count"`
	Similar     []int64 `json:"similar_games"`
	External    []struct {
		Source int    `json:"external_game_source"`
		UID    string `json:"uid"`
	} `json:"external_games"`
}

func (g igdbGame) toGame() Game {
	out := Game{
		IGDBID:      g.ID,
		Title:       g.Name,
		Platforms:   g.Platforms,
		Genres:      names(g.Genres),
		Themes:      names(g.Themes),
		Keywords:    names(g.Keywords),
		Rating:      g.Rating,
		RatingCount: g.RatingCount,
		SimilarIDs:  g.Similar,
	}
	if g.Released > 0 {
		out.ReleaseYear = time.Unix(g.Released, 0).UTC().Year()
	}
	if g.Cover != nil && g.Cover.ImageID != "" {
		out.CoverURL = fmt.Sprintf(coverURL, g.Cover.ImageID)
	}
	for _, c := range g.Companies {
		if c.Developer {
			out.Developers = append(out.Developers, c.Company.Name)
		}
	}
	for _, e := range g.External {
		if e.Source == sourceSteam && out.SteamAppID == "" {
			out.SteamAppID = e.UID
		}
	}
	return out
}

func names(ns []named) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = n.Name
	}
	return out
}

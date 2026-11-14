package stores

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	steamBaseURL   = "https://store.steampowered.com"
	steamCountry   = "in"
	steamBatchSize = 25
)

// SteamClient prices Steam apps in the India store.
type SteamClient struct {
	HTTP    *http.Client
	BaseURL string // overridden in tests
}

func (c *SteamClient) base() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return steamBaseURL
}

// ListingFor builds the Steam listing for a known appid (IGDB supplies it).
func (c *SteamClient) ListingFor(appid, title string) Listing {
	return Listing{
		Store:    Steam,
		Ref:      appid,
		Title:    title,
		URL:      "https://store.steampowered.com/app/" + appid,
		Region:   "IN",
		Currency: "INR",
	}
}

// Search finds Steam apps by name. Used only when IGDB has no Steam appid.
func (c *SteamClient) Search(ctx context.Context, term string) ([]Listing, error) {
	q := url.Values{"term": {term}, "cc": {steamCountry}, "l": {"english"}}
	var resp struct {
		Items []struct {
			Type string `json:"type"`
			Name string `json:"name"`
			ID   int64  `json:"id"`
		} `json:"items"`
	}
	if err := getJSON(ctx, orDefault(c.HTTP), c.base()+"/api/storesearch/?"+q.Encode(), &resp); err != nil {
		return nil, fmt.Errorf("steam search: %w", err)
	}
	out := make([]Listing, 0, len(resp.Items))
	for _, it := range resp.Items {
		if it.Type != "app" {
			continue
		}
		out = append(out, c.ListingFor(strconv.FormatInt(it.ID, 10), it.Name))
	}
	return out, nil
}

// Quotes returns the current price for each appid that Steam sells. Apps that
// are free, unreleased or delisted are absent from the map.
func (c *SteamClient) Quotes(ctx context.Context, appids []string) (map[string]Quote, error) {
	out := make(map[string]Quote, len(appids))
	now := time.Now().UTC()
	for _, batch := range chunk(appids, steamBatchSize) {
		q := url.Values{"appids": {strings.Join(batch, ",")}, "cc": {steamCountry}, "filters": {"price_overview"}}
		var resp map[string]struct {
			Success bool            `json:"success"`
			Data    json.RawMessage `json:"data"`
		}
		if err := getJSON(ctx, orDefault(c.HTTP), c.base()+"/api/appdetails?"+q.Encode(), &resp); err != nil {
			return nil, fmt.Errorf("steam appdetails: %w", err)
		}
		for appid, entry := range resp {
			quote, ok := parseSteamPrice(entry.Success, entry.Data)
			if !ok {
				continue
			}
			quote.Ref, quote.CheckedAt = appid, now
			out[appid] = quote
		}
	}
	return out, nil
}

// parseSteamPrice handles Steam's habit of returning `"data": []` instead of an
// object when an app has no price (free-to-play, not yet on sale).
func parseSteamPrice(success bool, data json.RawMessage) (Quote, bool) {
	if !success || len(data) == 0 || data[0] != '{' {
		return Quote{}, false
	}
	var d struct {
		Price *struct {
			Currency string `json:"currency"`
			Initial  int64  `json:"initial"`
			Final    int64  `json:"final"`
			Discount int    `json:"discount_percent"`
		} `json:"price_overview"`
	}
	if json.Unmarshal(data, &d) != nil || d.Price == nil {
		return Quote{}, false
	}
	return Quote{
		Currency:    d.Price.Currency,
		BaseMinor:   d.Price.Initial,
		FinalMinor:  d.Price.Final,
		DiscountPct: d.Price.Discount,
	}, true
}

func chunk[T any](xs []T, n int) [][]T {
	var out [][]T
	for len(xs) > n {
		out = append(out, xs[:n])
		xs = xs[n:]
	}
	if len(xs) > 0 {
		out = append(out, xs)
	}
	return out
}

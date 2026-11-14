package stores

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	eshopPriceURL  = "https://api.ec.nintendo.com"
	eshopCountry   = "US"
	eshopBatchSize = 50

	// nintendo.com's own storefront searches this Algolia index from the
	// browser with a search-only key. Both values are public, shipped in the
	// site's JavaScript; they can change whenever Nintendo rotates them.
	eshopAlgoliaURL   = "https://U3B6GR4UA3-dsn.algolia.net"
	eshopAlgoliaApp   = "U3B6GR4UA3"
	eshopAlgoliaKey   = "a29c6927638bfd8cee23993e51e721c9"
	eshopAlgoliaIndex = "store_game_en_us"
)

var eshopEditions = map[string]string{
	"NINTENDO_SWITCH_2": "Switch 2",
	"NINTENDO_SWITCH":   "Switch",
}

// EshopClient finds and prices games in the US Nintendo eShop.
type EshopClient struct {
	HTTP       *http.Client
	PriceURL   string // overridden in tests
	AlgoliaURL string // overridden in tests
}

// EshopHit is one Algolia search result.
type EshopHit struct {
	Title        string `json:"title"`
	NSUID        string `json:"nsuid"`
	PlatformCode string `json:"platformCode"`
	Category     string `json:"topLevelCategoryCode"`
	IsUpgrade    bool   `json:"isUpgrade"`
	URL          string `json:"url"`
	// PopularityRank is nintendo.com's ranking, 1 = most popular; 0 when unranked.
	PopularityRank int `json:"popularityRank"`
}

// Search returns raw eShop search hits for a title.
func (c *EshopClient) Search(ctx context.Context, title string) ([]EshopHit, error) {
	return c.query(ctx, url.Values{"query": {title}, "hitsPerPage": {"20"}})
}

// eshopDealsPage is as many hits as one Algolia query returns here.
const eshopDealsPage = "500"

// Deals returns games on sale in the eShop, most popular first, at most
// limit of them. DLC and upgrade packs are left out; Quotes prices them.
func (c *EshopClient) Deals(ctx context.Context, limit int) ([]Listing, error) {
	hits, err := c.query(ctx, url.Values{
		"query": {""}, "hitsPerPage": {eshopDealsPage}, "filters": {`topLevelFilters:"Deals"`},
	})
	if err != nil {
		return nil, err
	}
	rank := func(h EshopHit) int {
		if h.PopularityRank == 0 {
			return math.MaxInt
		}
		return h.PopularityRank
	}
	slices.SortStableFunc(hits, func(a, b EshopHit) int { return rank(a) - rank(b) })
	var out []Listing
	for _, h := range hits {
		edition, ok := eshopEditions[h.PlatformCode]
		if !ok || h.IsUpgrade || h.Category != "GAMES" || h.NSUID == "" {
			continue
		}
		out = append(out, Listing{
			Store: Eshop, Ref: h.NSUID, Title: h.Title, Edition: edition,
			URL: "https://www.nintendo.com" + h.URL, Region: eshopCountry, Currency: "USD",
		})
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (c *EshopClient) query(ctx context.Context, params url.Values) ([]EshopHit, error) {
	body, _ := json.Marshal(map[string]any{
		"requests": []map[string]string{{"indexName": eshopAlgoliaIndex, "params": params.Encode()}},
	})
	base := eshopAlgoliaURL
	if c.AlgoliaURL != "" {
		base = c.AlgoliaURL
	}
	req, err := http.NewRequest(http.MethodPost, base+"/1/indexes/*/queries", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Algolia-Application-Id", eshopAlgoliaApp)
	req.Header.Set("X-Algolia-API-Key", eshopAlgoliaKey)
	req.Header.Set("Content-Type", "application/json")
	var resp struct {
		Results []struct {
			Hits []EshopHit `json:"hits"`
		} `json:"results"`
	}
	if err := doJSON(ctx, orDefault(c.HTTP), req, &resp); err != nil {
		return nil, fmt.Errorf("eshop search: %w", err)
	}
	if len(resp.Results) == 0 {
		return nil, nil
	}
	return resp.Results[0].Hits, nil
}

// Find returns the eShop listings (at most one per Switch generation) whose
// title matches exactly after normalisation.
func (c *EshopClient) Find(ctx context.Context, title string) ([]Listing, error) {
	hits, err := c.Search(ctx, title)
	if err != nil {
		return nil, err
	}
	return MatchEshop(title, hits), nil
}

// MatchEshop picks the base game for each Switch generation from search hits,
// skipping DLC, upgrade packs and titles that don't match. An exact title wins
// over a named edition ("Elden Ring Tarnished Edition") of the same game.
func MatchEshop(title string, hits []EshopHit) []Listing {
	want := NormalizeTitle(title)
	best := map[string]EshopHit{}
	for _, h := range hits {
		edition, ok := eshopEditions[h.PlatformCode]
		if !ok || h.IsUpgrade || h.Category != "GAMES" || h.NSUID == "" {
			continue
		}
		score := MatchScore(want, NormalizeTitle(h.Title))
		if prev, seen := best[edition]; score == 0 || (seen && MatchScore(want, NormalizeTitle(prev.Title)) >= score) {
			continue
		}
		best[edition] = h
	}
	var out []Listing
	for _, h := range hits { // keep search order for stable output
		edition := eshopEditions[h.PlatformCode]
		if b, ok := best[edition]; !ok || b.NSUID != h.NSUID {
			continue
		}
		out = append(out, Listing{
			Store:    Eshop,
			Ref:      h.NSUID,
			Title:    h.Title,
			Edition:  edition,
			URL:      "https://www.nintendo.com" + h.URL,
			Region:   eshopCountry,
			Currency: "USD",
		})
	}
	return out
}

type eshopAmount struct {
	Currency string `json:"currency"`
	RawValue string `json:"raw_value"`
	End      string `json:"end_datetime"`
}

// Quotes returns the current price for each purchasable nsuid.
func (c *EshopClient) Quotes(ctx context.Context, nsuids []string) (map[string]Quote, error) {
	base := eshopPriceURL
	if c.PriceURL != "" {
		base = c.PriceURL
	}
	out := make(map[string]Quote, len(nsuids))
	now := time.Now().UTC()
	for _, batch := range chunk(nsuids, eshopBatchSize) {
		q := url.Values{"country": {eshopCountry}, "lang": {"en"}, "ids": {strings.Join(batch, ",")}}
		var resp struct {
			Prices []struct {
				TitleID  int64        `json:"title_id"`
				Status   string       `json:"sales_status"`
				Regular  *eshopAmount `json:"regular_price"`
				Discount *eshopAmount `json:"discount_price"`
			} `json:"prices"`
		}
		if err := getJSON(ctx, orDefault(c.HTTP), base+"/v1/price?"+q.Encode(), &resp); err != nil {
			return nil, fmt.Errorf("eshop price: %w", err)
		}
		for _, p := range resp.Prices {
			if p.Status != "onsale" || p.Regular == nil {
				continue
			}
			quote, err := eshopQuote(p.Regular, p.Discount)
			if err != nil {
				return nil, fmt.Errorf("eshop price %d: %w", p.TitleID, err)
			}
			quote.Ref, quote.CheckedAt = strconv.FormatInt(p.TitleID, 10), now
			out[quote.Ref] = quote
		}
	}
	return out, nil
}

func eshopQuote(regular, discount *eshopAmount) (Quote, error) {
	base, err := parseMinor(regular.RawValue)
	if err != nil {
		return Quote{}, err
	}
	q := Quote{Currency: regular.Currency, BaseMinor: base, FinalMinor: base}
	if discount == nil {
		return q, nil
	}
	if q.FinalMinor, err = parseMinor(discount.RawValue); err != nil {
		return Quote{}, err
	}
	q.DiscountPct = discountPct(q.BaseMinor, q.FinalMinor)
	if end, err := time.Parse(time.RFC3339, discount.End); err == nil {
		q.SaleEndsAt = &end
	}
	return q, nil
}

// parseMinor converts a decimal string like "79.99" to minor units (7999)
// without going through float64.
func parseMinor(s string) (int64, error) {
	whole, frac, _ := strings.Cut(strings.TrimSpace(s), ".")
	frac = (frac + "00")[:2]
	n, err := strconv.ParseInt(whole+frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse price %q: %w", s, err)
	}
	return n, nil
}

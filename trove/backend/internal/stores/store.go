// Package stores fetches listings and prices from the storefronts Trove tracks.
//
// Steam (India, INR) and the Nintendo eShop (US, USD) are priced. The PS Store
// is link-only: Sony has no public price API, so Trove links to the store page
// and prices are read there through the PlatPrices browser extension.
package stores

import "time"

const (
	Steam = "steam"
	Eshop = "eshop"
	PSN   = "psn"
)

// Listing is one purchasable version of a game in one store.
type Listing struct {
	Store    string
	Ref      string // Steam appid, eShop nsuid, or PS Store product id ("" for a search link)
	Title    string
	Edition  string // e.g. "Switch 2", "Switch"; empty when the store has one version
	URL      string
	Region   string
	Currency string
}

// Quote is a price observation for a listing. Amounts are in minor units
// (paise, cents) so comparisons never touch floating point.
type Quote struct {
	Ref         string
	Currency    string
	BaseMinor   int64
	FinalMinor  int64
	DiscountPct int
	SaleEndsAt  *time.Time
	CheckedAt   time.Time
}

// OnSale reports whether the quote is discounted.
func (q Quote) OnSale() bool { return q.FinalMinor < q.BaseMinor }

func discountPct(base, final int64) int {
	if base <= 0 || final >= base {
		return 0
	}
	return int((base - final) * 100 / base)
}

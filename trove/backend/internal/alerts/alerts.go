// Package alerts decides when a price change is worth telling you about.
package alerts

import (
	"fmt"

	"github.com/saifkazi/trove/backend/internal/stores"
)

// Reasons a sale event is raised. One event can carry several.
const (
	ReasonSale   = "sale"   // newly discounted, or discounted further
	ReasonTarget = "target" // dropped to or below your target price
	ReasonLowest = "lowest" // cheapest Trove has ever seen it
)

// Check compares a new quote with what Trove last saw for the listing.
type Check struct {
	New stores.Quote
	// Previous is the last recorded quote; nil on the first check.
	Previous *stores.Quote
	// LowestMinor is the lowest final price recorded before New, 0 if none.
	LowestMinor int64
	// TargetINR is the wishlist target in whole rupees, if set.
	TargetINR *int32
	// ToINRMinor converts an amount in the quote's currency to paise.
	ToINRMinor func(minor int64) int64
}

// Evaluate returns why the change matters (empty when it doesn't) and a key
// identifying this particular price, so the same sale never alerts twice.
// Only a change in price can raise an event: exchange-rate drift alone, or
// seeing the same price again, never does.
func Evaluate(c Check) (reasons []string, saleKey string) {
	if c.Previous != nil && c.Previous.FinalMinor == c.New.FinalMinor {
		return nil, ""
	}
	// A sale only counts as news when it starts or gets deeper; a sale that
	// becomes less generous is a price rise.
	startedOrDeeper := c.Previous == nil || !c.Previous.OnSale() || c.New.FinalMinor < c.Previous.FinalMinor
	if c.New.OnSale() && startedOrDeeper {
		reasons = append(reasons, ReasonSale)
	}
	if c.TargetINR != nil && c.ToINRMinor != nil {
		target := int64(*c.TargetINR) * 100
		wasAbove := c.Previous == nil || c.ToINRMinor(c.Previous.FinalMinor) > target
		if wasAbove && c.ToINRMinor(c.New.FinalMinor) <= target {
			reasons = append(reasons, ReasonTarget)
		}
	}
	if c.LowestMinor > 0 && c.New.FinalMinor < c.LowestMinor {
		reasons = append(reasons, ReasonLowest)
	}
	if len(reasons) == 0 {
		return nil, ""
	}
	return reasons, SaleKey(c.New)
}

// SaleKey identifies a price within a sale window: the same discounted price
// in the same sale gets the same key.
func SaleKey(q stores.Quote) string {
	ends := "open"
	if q.SaleEndsAt != nil {
		ends = q.SaleEndsAt.UTC().Format("2006-01-02")
	}
	return fmt.Sprintf("%d:%s", q.FinalMinor, ends)
}

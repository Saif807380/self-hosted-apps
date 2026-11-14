package alerts

import (
	"slices"
	"testing"
	"time"

	"github.com/saifkazi/trove/backend/internal/stores"
)

func quote(base, final int64) stores.Quote {
	return stores.Quote{Currency: "INR", BaseMinor: base, FinalMinor: final}
}

func ptr[T any](v T) *T { return &v }

func identity(m int64) int64 { return m }

func TestEvaluate(t *testing.T) {
	usdToINR := func(m int64) int64 { return m * 94 } // cents -> paise at ₹94/$
	cases := []struct {
		name string
		c    Check
		want []string
	}{
		{"first check at full price", Check{New: quote(130000, 130000)}, nil},
		{"first check already on sale", Check{New: quote(130000, 91000)}, []string{ReasonSale}},
		{"same price again", Check{New: quote(130000, 91000), Previous: ptr(quote(130000, 91000)), LowestMinor: 91000}, nil},
		{"sale starts", Check{New: quote(130000, 91000), Previous: ptr(quote(130000, 130000)), LowestMinor: 130000},
			[]string{ReasonSale, ReasonLowest}},
		{"deeper discount", Check{New: quote(130000, 65000), Previous: ptr(quote(130000, 91000)), LowestMinor: 91000},
			[]string{ReasonSale, ReasonLowest}},
		{"sale gets shallower", Check{New: quote(130000, 91000), Previous: ptr(quote(130000, 65000)), LowestMinor: 65000}, nil},
		{"sale ends", Check{New: quote(130000, 130000), Previous: ptr(quote(130000, 91000)), LowestMinor: 91000}, nil},
		{"sale returns, not a new low", Check{New: quote(130000, 91000), Previous: ptr(quote(130000, 130000)), LowestMinor: 65000},
			[]string{ReasonSale}},
		{"base price cut to target", Check{New: quote(100000, 100000), Previous: ptr(quote(130000, 130000)),
			LowestMinor: 130000, TargetINR: ptr(int32(1000)), ToINRMinor: identity}, []string{ReasonTarget, ReasonLowest}},
		{"already below target", Check{New: quote(130000, 80000), Previous: ptr(quote(130000, 90000)),
			LowestMinor: 90000, TargetINR: ptr(int32(1000)), ToINRMinor: identity}, []string{ReasonSale, ReasonLowest}},
		{"eShop USD price under rupee target", Check{
			New: stores.Quote{Currency: "USD", BaseMinor: 1999, FinalMinor: 999}, Previous: &stores.Quote{Currency: "USD", BaseMinor: 1999, FinalMinor: 1999},
			LowestMinor: 1999, TargetINR: ptr(int32(1000)), ToINRMinor: usdToINR}, []string{ReasonSale, ReasonTarget, ReasonLowest}},
	}
	for _, c := range cases {
		got, key := Evaluate(c.c)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: reasons = %v, want %v", c.name, got, c.want)
		}
		if (key == "") != (len(c.want) == 0) {
			t.Errorf("%s: key = %q with reasons %v", c.name, key, got)
		}
	}
}

func TestSaleKey(t *testing.T) {
	end := time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC)
	q := quote(1999, 499)
	if got := SaleKey(q); got != "499:open" {
		t.Errorf("open sale key = %s", got)
	}
	q.SaleEndsAt = &end
	if got := SaleKey(q); got != "499:2026-11-30" {
		t.Errorf("dated sale key = %s", got)
	}
}

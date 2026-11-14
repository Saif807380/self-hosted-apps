package tracker

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/saifkazi/trove/backend/internal/alerts"
	"github.com/saifkazi/trove/backend/internal/dao"
)

func TestFormatSale(t *testing.T) {
	id := uuid.MustParse("6264ef12-3efe-4065-b637-5ed865fa1ff1")
	cover := "/uploads/hades.jpg"
	target := int32(2000)
	pct := func(n int16) *int16 { return &n }
	n := FormatSale([]dao.PendingSale{
		{GameID: id, GameTitle: "Hades II", CoverImage: &cover, TargetINR: &target, Store: "steam", Currency: "INR",
			Reasons: []string{alerts.ReasonSale, alerts.ReasonLowest}, FinalMinor: 91000, DiscountPct: pct(30)},
		{GameID: id, GameTitle: "Hades II", CoverImage: &cover, TargetINR: &target, Store: "eshop", Edition: "Switch 2", Currency: "USD",
			Reasons: []string{alerts.ReasonSale, alerts.ReasonTarget}, FinalMinor: 1999, DiscountPct: pct(33)},
	}, 94.25)

	if n.Title != "Hades II is on sale" {
		t.Errorf("title = %q", n.Title)
	}
	want := "Steam ₹910 (−30%) · lowest yet\neShop Switch 2 $19.99 (≈₹1,884) (−33%) · under your ₹2,000 target"
	if n.Body != want {
		t.Errorf("body =\n%s\nwant\n%s", n.Body, want)
	}
	if n.URL != "/games?tab=wishlist&game="+id.String() || n.Icon != cover {
		t.Errorf("url/icon = %s %s", n.URL, n.Icon)
	}
	if len(n.Tag) != 32 || strings.Contains(n.Tag, "-") {
		t.Errorf("tag %q isn't a valid push topic", n.Tag)
	}

	// A base-price cut with no discount reads as a drop, not a sale.
	drop := FormatSale([]dao.PendingSale{{GameID: id, GameTitle: "Hades II", Store: "steam", Currency: "INR",
		Reasons: []string{alerts.ReasonLowest}, FinalMinor: 100000}}, 0)
	if drop.Title != "Hades II dropped in price" || drop.Icon != "/icon-192.png" || drop.Body != "Steam ₹1,000 · lowest yet" {
		t.Errorf("drop = %+v", drop)
	}
}

func TestFormatINR(t *testing.T) {
	for in, want := range map[int64]string{0: "₹0", 999: "₹999", 1300: "₹1,300", 130000: "₹1,30,000", 12345678: "₹1,23,45,678"} {
		if got := formatINR(in); got != want {
			t.Errorf("formatINR(%d) = %s, want %s", in, got, want)
		}
	}
}

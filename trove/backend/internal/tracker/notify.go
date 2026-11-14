package tracker

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/saifkazi/trove/backend/internal/alerts"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/push"
	"github.com/saifkazi/trove/backend/internal/stores"
)

// saleAlertMaxAge drops sale events nobody could be told about in time (no
// subscription, keys missing): a sale from last week isn't news, and turning
// alerts on shouldn't bring a backlog.
const saleAlertMaxAge = 3 * 24 * time.Hour

// Notifier sends a notification to every subscribed browser.
type Notifier interface {
	Enabled() bool
	Send(ctx context.Context, n push.Notification) (delivered int, err error)
}

// DeliverSales turns pending sale events into one notification per game.
// Events stay pending until at least one browser accepts the notification,
// so a phone that subscribes within saleAlertMaxAge still hears about them.
func (t *Tracker) DeliverSales(ctx context.Context) error {
	if t.Notify == nil || !t.Notify.Enabled() {
		return nil
	}
	pending, err := t.Store.PendingSales(ctx)
	if err != nil || len(pending) == 0 {
		return err
	}

	var skipped []int64
	var order []uuid.UUID
	byGame := map[uuid.UUID][]dao.PendingSale{}
	for _, p := range pending {
		if !p.StillWanted || time.Since(p.CreatedAt) > saleAlertMaxAge {
			skipped = append(skipped, p.ID)
			continue
		}
		if _, seen := byGame[p.GameID]; !seen {
			order = append(order, p.GameID)
		}
		byGame[p.GameID] = append(byGame[p.GameID], p)
	}
	if err := t.Store.MarkSalesNotified(ctx, skipped); err != nil {
		return err
	}

	usdToINR, _, err := t.Store.FXRate(ctx, "USD", "INR")
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	for _, gameID := range order {
		sales := byGame[gameID]
		delivered, err := t.Notify.Send(ctx, FormatSale(sales, usdToINR))
		if err != nil {
			return err
		}
		if delivered == 0 {
			continue // no subscribed browser took it; try again next run
		}
		ids := make([]int64, len(sales))
		for i, s := range sales {
			ids[i] = s.ID
		}
		if err := t.Store.MarkSalesNotified(ctx, ids); err != nil {
			return err
		}
		t.Log.Info("sale notification sent", "game", sales[0].GameTitle, "devices", delivered)
	}
	return nil
}

// FormatSale writes the notification for one game's sales, one line per
// store:
//
//	Hades II is on sale
//	Steam ₹910 (−30%) · lowest yet
//	eShop Switch 2 $19.99 (≈₹1,884) (−33%) · under your ₹2,000 target
func FormatSale(sales []dao.PendingSale, usdToINR float64) push.Notification {
	game := sales[0]
	onSale := false
	lines := make([]string, len(sales))
	for i, s := range sales {
		onSale = onSale || slices.Contains(s.Reasons, alerts.ReasonSale)
		line := storeLabel(s.Store, s.Edition) + " " + formatPrice(s.FinalMinor, s.Currency, usdToINR)
		if s.DiscountPct != nil && *s.DiscountPct > 0 {
			line += fmt.Sprintf(" (−%d%%)", *s.DiscountPct)
		}
		var notes []string
		if slices.Contains(s.Reasons, alerts.ReasonLowest) {
			notes = append(notes, "lowest yet")
		}
		if slices.Contains(s.Reasons, alerts.ReasonTarget) && game.TargetINR != nil {
			notes = append(notes, "under your "+formatINR(int64(*game.TargetINR))+" target")
		}
		if len(notes) > 0 {
			line += " · " + strings.Join(notes, " · ")
		}
		lines[i] = line
	}

	title := game.GameTitle + " is on sale"
	if !onSale {
		title = game.GameTitle + " dropped in price"
	}
	icon := "/icon-192.png"
	if game.CoverImage != nil {
		icon = *game.CoverImage
	}
	return push.Notification{
		Title: title,
		Body:  strings.Join(lines, "\n"),
		URL:   "/games?tab=wishlist&game=" + game.GameID.String(),
		Icon:  icon,
		// 32 characters: also valid as a push Topic, which caps at 32.
		Tag: strings.ReplaceAll(game.GameID.String(), "-", ""),
	}
}

func storeLabel(store, edition string) string {
	switch store {
	case stores.Steam:
		return "Steam"
	case stores.Eshop:
		return strings.TrimSpace("eShop " + edition)
	default:
		return store
	}
}

func formatPrice(minor int64, currency string, usdToINR float64) string {
	switch currency {
	case "INR":
		return formatINR(int64(math.Round(float64(minor) / 100)))
	case "USD":
		s := fmt.Sprintf("$%.2f", float64(minor)/100)
		if paise, ok := ToINRMinor(minor, currency, usdToINR); ok {
			s += " (≈" + formatINR(int64(math.Round(float64(paise)/100))) + ")"
		}
		return s
	default:
		return fmt.Sprintf("%.2f %s", float64(minor)/100, currency)
	}
}

// formatINR groups rupees the Indian way: ₹1,300 · ₹1,30,000.
func formatINR(rupees int64) string {
	digits := fmt.Sprint(rupees)
	if len(digits) <= 3 {
		return "₹" + digits
	}
	head, tail := digits[:len(digits)-3], digits[len(digits)-3:]
	var groups []string
	for len(head) > 2 {
		groups = append([]string{head[len(head)-2:]}, groups...)
		head = head[:len(head)-2]
	}
	groups = append([]string{head}, groups...)
	return "₹" + strings.Join(groups, ",") + "," + tail
}

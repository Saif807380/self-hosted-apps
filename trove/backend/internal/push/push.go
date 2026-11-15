// Package push sends web push notifications to the browsers that turned on
// sale alerts.
package push

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/saifkazi/trove/backend/internal/dao"
)

const (
	// ttlSeconds is how long a push service holds a notification for a phone
	// that's offline; a sale alert older than a day isn't worth delivering.
	ttlSeconds = 24 * 60 * 60

	// recordSize is the encrypted message size; every message is padded to
	// it. webpush-go defaults to 4096, which Mozilla's push service rejects
	// for phones ("413 … intended for a constrained device").
	recordSize = 2048
	// maxPayload leaves room in a record for the encryption header (86
	// bytes), the auth tag (16) and the padding delimiter.
	maxPayload = recordSize - 128
)

// Notification is the payload the service worker (ui/src/sw.ts) shows.
type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`            // opened when the notification is tapped
	Icon  string `json:"icon,omitempty"` // e.g. the game's cover
	Tag   string `json:"tag,omitempty"`  // a newer notification with the same tag replaces the older
}

// Subscriptions is the storage the sender needs.
type Subscriptions interface {
	PushSubscriptions(ctx context.Context) ([]dao.PushSubscription, error)
	DeletePushSubscription(ctx context.Context, endpoint string) error
	MarkPushOK(ctx context.Context, endpoint string) error
}

// Sender delivers notifications using VAPID keys.
type Sender struct {
	Store      Subscriptions
	PublicKey  string
	PrivateKey string
	Subject    string
	Log        *slog.Logger
	HTTP       webpush.HTTPClient // overridden in tests
}

// Enabled reports whether VAPID keys are configured.
func (s *Sender) Enabled() bool {
	return s.PublicKey != "" && s.PrivateKey != ""
}

// Send delivers n to every subscribed browser and returns how many accepted
// it. Subscriptions the push service reports as gone are deleted.
func (s *Sender) Send(ctx context.Context, n Notification) (int, error) {
	if !s.Enabled() {
		return 0, fmt.Errorf("push: VAPID keys not configured")
	}
	subs, err := s.Store.PushSubscriptions(ctx)
	if err != nil {
		return 0, err
	}
	payload, err := encode(n)
	if err != nil {
		return 0, err
	}
	delivered := 0
	for _, sub := range subs {
		ok, err := s.deliver(ctx, payload, sub, n.Tag)
		if err != nil {
			return delivered, err
		}
		if ok {
			delivered++
		}
	}
	return delivered, nil
}

func (s *Sender) deliver(ctx context.Context, payload []byte, sub dao.PushSubscription, tag string) (bool, error) {
	resp, err := webpush.SendNotificationWithContext(ctx, payload,
		&webpush.Subscription{Endpoint: sub.Endpoint, Keys: webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth}},
		&webpush.Options{
			HTTPClient:      s.HTTP,
			Subscriber:      s.Subject,
			VAPIDPublicKey:  s.PublicKey,
			VAPIDPrivateKey: s.PrivateKey,
			TTL:             ttlSeconds,
			RecordSize:      recordSize,
			// Every push shows a notification, which is what high urgency is
			// for; at normal, an idle Android phone holds it until Chrome wakes.
			Urgency: webpush.UrgencyHigh,
			Topic:   tag, // the push service also collapses undelivered ones
		})
	if err != nil {
		s.Log.Warn("push failed", "endpoint", host(sub.Endpoint), "err", err)
		return false, nil // network trouble with one service shouldn't stop the rest
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode/100 == 2:
		return true, s.Store.MarkPushOK(ctx, sub.Endpoint)
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		s.Log.Info("push subscription gone, removing", "endpoint", host(sub.Endpoint))
		return false, s.Store.DeletePushSubscription(ctx, sub.Endpoint)
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		s.Log.Warn("push rejected", "endpoint", host(sub.Endpoint), "status", resp.StatusCode, "body", string(body))
		return false, nil
	}
}

// encode marshals n, shortening the body if needed to fit in one record.
func encode(n Notification) ([]byte, error) {
	for {
		payload, err := json.Marshal(n)
		if err != nil || len(payload) <= maxPayload {
			return payload, err
		}
		runes := []rune(n.Body)
		if len(runes) == 0 {
			return nil, fmt.Errorf("push: notification too large even without a body")
		}
		cut := max(len(runes)-(len(payload)-maxPayload)-1, 0)
		n.Body = string(runes[:cut]) + "…"
	}
}

// host keeps logs free of the full endpoint, which works as a credential.
func host(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "?"
	}
	return u.Host
}

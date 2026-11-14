package push

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/saifkazi/trove/backend/internal/dao"
)

type fakeStore struct {
	subs    []dao.PushSubscription
	deleted []string
	ok      []string
}

func (f *fakeStore) PushSubscriptions(context.Context) ([]dao.PushSubscription, error) {
	return f.subs, nil
}
func (f *fakeStore) DeletePushSubscription(_ context.Context, e string) error {
	f.deleted = append(f.deleted, e)
	return nil
}
func (f *fakeStore) MarkPushOK(_ context.Context, e string) error { f.ok = append(f.ok, e); return nil }

// fakePushService answers by endpoint: "/gone" with 410, anything else 201.
type fakePushService struct{ requests []*http.Request }

func (f *fakePushService) Do(r *http.Request) (*http.Response, error) {
	f.requests = append(f.requests, r)
	status := http.StatusCreated
	if strings.HasSuffix(r.URL.Path, "/gone") {
		status = http.StatusGone
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}, nil
}

// A real browser key pair (from the web-push test vectors), so payload
// encryption runs for real.
const (
	testP256dh = "BNNL5ZaTfK81qhXOx23-wewhigUeFb632jN6LvRWCFH1ubQr77FE_9qV1FuojuRmHP42zmf34rXgW80OvUVDgTk"
	testAuth   = "zqbxT6JKstKSY9JKibZLSQ"
)

func TestSend(t *testing.T) {
	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{subs: []dao.PushSubscription{
		{Endpoint: "https://fcm.googleapis.com/fcm/send/phone", P256dh: testP256dh, Auth: testAuth},
		{Endpoint: "https://fcm.googleapis.com/fcm/send/gone", P256dh: testP256dh, Auth: testAuth},
	}}
	svc := &fakePushService{}
	s := &Sender{
		Store: store, PublicKey: public, PrivateKey: private, Subject: "https://trove.example",
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)), HTTP: svc,
	}

	delivered, err := s.Send(context.Background(), Notification{Title: "Hades II is on sale", URL: "/games", Tag: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if delivered != 1 {
		t.Errorf("delivered = %d, want 1", delivered)
	}
	if len(store.ok) != 1 || !strings.HasSuffix(store.ok[0], "/phone") {
		t.Errorf("marked ok = %v", store.ok)
	}
	if len(store.deleted) != 1 || !strings.HasSuffix(store.deleted[0], "/gone") {
		t.Errorf("deleted = %v, want the 410 subscription", store.deleted)
	}
	r := svc.requests[0]
	if r.Header.Get("TTL") != "86400" || r.Header.Get("Topic") != "abc" || !strings.HasPrefix(r.Header.Get("Authorization"), "vapid ") {
		t.Errorf("headers = %v", r.Header)
	}
	if r.Header.Get("Content-Encoding") != "aes128gcm" {
		t.Errorf("payload not encrypted: %v", r.Header)
	}
	// Mozilla rejects 4096-byte records for phones; ours must stay at 2048.
	if body, _ := io.ReadAll(r.Body); len(body) > recordSize {
		t.Errorf("encrypted body is %d bytes, want at most %d", len(body), recordSize)
	}
}

func TestEncodeShortensLongBodies(t *testing.T) {
	n := Notification{Title: "Hades II is on sale", Body: strings.Repeat("Steam ₹910 (−30%) · lowest yet\n", 200), URL: "/games"}
	payload, err := encode(n)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) > maxPayload || !strings.Contains(string(payload), "…") {
		t.Errorf("payload %d bytes (max %d), truncated=%t", len(payload), maxPayload, strings.Contains(string(payload), "…"))
	}
	short, _ := encode(Notification{Title: "t", Body: "short", URL: "/"})
	if strings.Contains(string(short), "…") {
		t.Error("short body was truncated")
	}
}

func TestSendWithoutKeys(t *testing.T) {
	s := &Sender{Store: &fakeStore{}}
	if s.Enabled() {
		t.Fatal("enabled without keys")
	}
	if _, err := s.Send(context.Background(), Notification{}); err == nil {
		t.Fatal("expected an error without keys")
	}
}

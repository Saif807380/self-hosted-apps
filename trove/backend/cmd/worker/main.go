// Command worker runs Trove's price tracking: it links games to IGDB, finds
// their store listings, checks prices and records sale events. It runs as its
// own container so slow store calls never hold up the API.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/config"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/jobs"
	"github.com/saifkazi/trove/backend/internal/push"
	"github.com/saifkazi/trove/backend/internal/stores"
	"github.com/saifkazi/trove/backend/internal/tracker"
)

const (
	linkEvery  = time.Hour
	priceEvery = 6 * time.Hour
	// How long one BLPOP blocks. A blocked BLPOP doesn't notice shutdown, so
	// this bounds how long stopping takes; podman stop kills after 10s.
	queueWait  = 5 * time.Second
	startDelay = 5 * time.Second // let Postgres and Redis settle after a reboot
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()
	if cfg.IGDBClientID == "" || cfg.IGDBClientSecret == "" {
		log.Error("IGDB_CLIENT_ID and IGDB_CLIENT_SECRET must be set")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := dao.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	rdb := dao.NewRedisClient(cfg.RedisURL)
	defer rdb.Close()

	gaming := dao.NewStores(pool).Gaming
	sender := &push.Sender{
		Store: gaming, PublicKey: cfg.VAPIDPublicKey, PrivateKey: cfg.VAPIDPrivateKey,
		Subject: cfg.VAPIDSubject, Log: log,
	}
	if !sender.Enabled() {
		log.Warn("VAPID keys not set; sale alerts will not be sent")
	}
	t := &tracker.Tracker{
		Store: gaming,
		IGDB:  &catalog.Client{ClientID: cfg.IGDBClientID, ClientSecret: cfg.IGDBClientSecret},
		Steam: &stores.SteamClient{},
		Eshop: &stores.EshopClient{},
		FX:    &stores.FXClient{},
		Log:   log,
		// Shared with the backend, which serves and syncs these as covers.
		UploadsDir: cfg.UploadsDir,
		Notify:     sender,
	}
	w := &worker{t: t, log: log}

	log.Info("worker started", "link_every", linkEvery, "price_every", priceEvery)
	select {
	case <-time.After(startDelay):
	case <-ctx.Done():
		return
	}

	var wg sync.WaitGroup
	wg.Go(func() { w.every(ctx, linkEvery, "link", w.link) })
	wg.Go(func() { w.every(ctx, priceEvery, "prices", w.prices) })
	wg.Go(func() { w.consume(ctx, rdb) })
	wg.Wait()
	log.Info("worker stopped")
}

// worker serialises the pipeline steps: scheduled runs and on-demand
// refreshes touch the same rows, and there's no need to run them in parallel.
type worker struct {
	t   *tracker.Tracker
	log *slog.Logger
	mu  sync.Mutex
}

func (w *worker) link(ctx context.Context) error {
	return w.t.LinkPending(ctx)
}

func (w *worker) prices(ctx context.Context) error {
	if err := w.t.RefreshFX(ctx); err != nil {
		w.log.Warn("fx refresh failed", "err", err)
	}
	if err := w.t.FindListings(ctx, nil); err != nil {
		return err
	}
	if err := w.t.CheckPrices(ctx, nil); err != nil {
		return err
	}
	return w.t.DeliverSales(ctx)
}

// every runs job now and then on each tick until ctx is cancelled.
func (w *worker) every(ctx context.Context, interval time.Duration, name string, job func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		w.run(ctx, name, job)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *worker) run(ctx context.Context, name string, job func(context.Context) error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	start := time.Now()
	if err := job(ctx); err != nil && ctx.Err() == nil {
		w.log.Error("job failed", "job", name, "err", err)
		return
	}
	w.log.Info("job done", "job", name, "took", time.Since(start).Round(time.Millisecond))
}

// consume handles refresh requests queued by the API.
func (w *worker) consume(ctx context.Context, rdb *redis.Client) {
	for ctx.Err() == nil {
		id, ok, err := jobs.NextRefresh(ctx, rdb, queueWait)
		if err != nil {
			if ctx.Err() == nil {
				w.log.Warn("refresh queue", "err", err)
				time.Sleep(queueWait)
			}
			continue
		}
		if ok {
			w.run(ctx, "refresh "+id.String(), func(ctx context.Context) error { return w.t.Refresh(ctx, id) })
		}
	}
}

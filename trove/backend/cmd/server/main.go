package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/config"
	"github.com/saifkazi/trove/backend/internal/controller"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/push"
	"github.com/saifkazi/trove/backend/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := dao.NewPostgresPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	redisClient := dao.NewRedisClient(cfg.RedisURL)
	defer redisClient.Close()

	stores := dao.NewStores(db)
	igdb := &catalog.Client{ClientID: cfg.IGDBClientID, ClientSecret: cfg.IGDBClientSecret}
	sender := &push.Sender{
		Store: stores.Gaming, PublicKey: cfg.VAPIDPublicKey, PrivateKey: cfg.VAPIDPrivateKey,
		Subject: cfg.VAPIDSubject, Log: slog.Default(),
	}
	services := service.NewServices(stores, redisClient, igdb, sender, cfg.AnthropicAPIKey != "")
	mux := controller.NewRouter(services, cfg.UploadsDir, cfg.TLSEnabled())

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if cfg.TLSEnabled() {
			log.Printf("server listening on :%d (TLS)", cfg.Port)
			if err := srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("server error: %v", err)
			}
		} else {
			log.Printf("server listening on :%d", cfg.Port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("server error: %v", err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("server stopped")
}

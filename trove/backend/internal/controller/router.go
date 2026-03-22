package controller

import (
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/saifkazi/trove/backend/gen/trove/v1/books/booksv1connect"
	"github.com/saifkazi/trove/backend/gen/trove/v1/games/gamesv1connect"
	"github.com/saifkazi/trove/backend/gen/trove/v1/sync/syncv1connect"
	"github.com/saifkazi/trove/backend/gen/trove/v1/travel/travelv1connect"
	"github.com/saifkazi/trove/backend/gen/trove/v1/workouts/workoutsv1connect"
	"github.com/saifkazi/trove/backend/internal/service"
)

func NewRouter(services *service.Services, uploadsDir string) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	// Serve uploaded images
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))))

	// REST upload endpoint
	mux.HandleFunc("/v1/uploads", handleUpload(uploadsDir))

	// Connect RPC service handlers
	mux.Handle(booksv1connect.NewBookServiceHandler(&bookHandler{svc: services.Books}))
	mux.Handle(gamesv1connect.NewGameServiceHandler(&gameHandler{svc: services.Games}))
	mux.Handle(travelv1connect.NewTravelServiceHandler(&travelHandler{svc: services.Travel}))
	mux.Handle(workoutsv1connect.NewWorkoutServiceHandler(&workoutHandler{svc: services.Workouts}))
	mux.Handle(syncv1connect.NewSyncServiceHandler(
		&syncHandler{svc: services.Sync},
		connect.WithReadMaxBytes(32<<20),
	))

	// Sync image REST endpoints
	mux.HandleFunc("/sync/images", handleSyncImageUpload(uploadsDir))
	mux.HandleFunc("/sync/images/manifest", handleSyncImageManifest(uploadsDir))

	// h2c enables HTTP/2 over cleartext (needed for gRPC protocol support)
	return h2c.NewHandler(withCORS(mux), &http2.Server{})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, Grpc-Timeout, X-Grpc-Web")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"

	"pulse-backend/internal/config"
	"pulse-backend/internal/features/alerts"
	"pulse-backend/internal/features/auth"
	"pulse-backend/internal/features/devices"
	"pulse-backend/internal/features/readings"
	"pulse-backend/internal/platform/database"
	"pulse-backend/internal/platform/httpx"
	"pulse-backend/internal/platform/realtime"
	"pulse-backend/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.DB.DSN(), 60*time.Second)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	log.Println("connected to MySQL")

	if err := database.Migrate(ctx, db, migrations.FS); err != nil {
		log.Fatalf("database: %v", err)
	}

	hub := realtime.NewHub()
	origins := httpx.NewOriginPolicy(cfg.HTTP.CORSOrigins)

	tokens := auth.NewTokenIssuer(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL)
	authService := auth.NewService(auth.NewMySQLRepository(db), tokens, cfg.Auth.RefreshTTL)
	authHandler := auth.NewHandler(authService, tokens)

	deviceService := devices.NewService(db)

	alertService := alerts.NewService(alerts.NewMySQLRepository(db), hub)
	readingService := readings.NewService(readings.NewMySQLRepository(db), deviceService, alertService, hub)
	alertService.SetLatestReader(readingService)

	subscriber := readings.NewMQTTSubscriber(readings.SubscriberConfig{
		BrokerURL: cfg.MQTT.BrokerURL,
		Username:  cfg.MQTT.Username,
		Password:  cfg.MQTT.Password,
		ClientID:  cfg.MQTT.ClientID,
	}, readingService)
	subscriber.Start()
	defer subscriber.Stop()

	limiter := httpx.NewRateLimiter(6*time.Second, 10, cfg.HTTP.TrustProxy) // ~10 auth attempts/min per IP
	requireAuth := authHandler.RequireAuth

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", healthz(db))
	mux.HandleFunc("GET /api/ws", realtime.Handler(hub, tokens, origins))
	authHandler.Register(mux, limiter.Limit)
	devices.NewHandler(deviceService, summaries{readings: readingService, alerts: alertService}).Register(mux, requireAuth)
	readings.NewHandler(readingService, deviceService).Register(mux, requireAuth)
	alerts.NewHandler(alertService, deviceService).Register(mux, requireAuth)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           httpx.Recover(httpx.Logger(origins.CORS(mux))),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on %s (CORS origins: %v)", srv.Addr, cfg.HTTP.CORSOrigins)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server failed: %v", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	hub.CloseAll() // hijacked WebSocket connections are not closed by Shutdown
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}
}

func healthz(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "database unavailable"})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// summaries feeds the device list with each device's latest reading and
// open alert count.
type summaries struct {
	readings *readings.Service
	alerts   *alerts.Service
}

func (s summaries) Summary(ctx context.Context, deviceID string) (devices.Summary, error) {
	latest, err := s.readings.Latest(ctx, deviceID)
	if err != nil {
		return devices.Summary{}, err
	}
	open, err := s.alerts.CountOpen(ctx, deviceID)
	if err != nil {
		return devices.Summary{}, err
	}
	sum := devices.Summary{OpenAlerts: open}
	if latest != nil { // avoid a typed-nil pointer inside the interface
		sum.Latest = latest
	}
	return sum, nil
}

package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"hydro-backend/internal/config"
	"hydro-backend/internal/features/readings"
	"hydro-backend/internal/platform/database"
	"hydro-backend/internal/platform/httpmiddleware"
	"hydro-backend/internal/platform/realtime"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.NewDB(cfg.DB.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("✅ connected to MySQL successfully")

	readingRepo := readings.NewMySQLRepository(db)
	readingService := readings.NewService(readingRepo)
	readingHandler := readings.NewHTTPHandler(readingService)

	hub := realtime.NewHub()
	readingService.SetBroadcaster(hub)

	subscriber := readings.NewMQTTSubscriber(cfg.MQTT.BrokerURL, readingService)
	if err := subscriber.Start(); err != nil {
		log.Fatalf("failed to start MQTT subscriber: %v", err)
	}
	defer subscriber.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /readings", readingHandler.CreateReading)
	mux.HandleFunc("GET /readings", readingHandler.GetHistory)
	mux.HandleFunc("GET /readings/latest", readingHandler.GetLatest)
	mux.HandleFunc("GET /ws", realtime.Handler(hub))

	loggedMux := httpmiddleware.Logger(mux)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: loggedMux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("🚀 server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("🛑 shutdown signal received, draining in-flight requests...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ forced shutdown: %v", err)
	} else {
		log.Println("✅ server shut down cleanly")
	}
}

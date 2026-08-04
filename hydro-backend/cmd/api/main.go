// cmd/api/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"hydro-backend/internal/config"
	"hydro-backend/internal/handler"
	"hydro-backend/internal/mqtt"
	"hydro-backend/internal/repository"
	"hydro-backend/internal/service"
	"hydro-backend/internal/websocket"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := repository.NewDB(cfg.DB.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("✅ connected to MySQL successfully")

	readingRepo := repository.NewMySQLReadingRepository(db)
	readingService := service.NewReadingService(readingRepo)
	readingHandler := handler.NewReadingHandler(readingService)

	// WebSocket hub — created once, shared by the /ws route (which
	// registers new client connections) and the reading service (which
	// broadcasts to those clients after every successful insert,
	// regardless of whether the reading came from HTTP or MQTT).
	hub := websocket.NewHub()
	readingService.SetBroadcaster(hub)

	// MQTT subscriber shares the exact same readingService instance as the
	// HTTP handler above — this is the whole point of the layered
	// architecture. Whether a reading arrives via HTTP POST or MQTT
	// publish, it goes through identical validation and storage logic.
	subscriber := mqtt.NewSubscriber(cfg.MQTT.BrokerURL, readingService)
	if err := subscriber.Start(); err != nil {
		log.Fatalf("failed to start MQTT subscriber: %v", err)
	}
	defer subscriber.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /readings", readingHandler.CreateReading)
	mux.HandleFunc("GET /readings", readingHandler.GetHistory)
	mux.HandleFunc("GET /readings/latest", readingHandler.GetLatest)
	mux.HandleFunc("GET /ws", websocket.Handler(hub))

	loggedMux := handler.Logger(mux)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: loggedMux,
	}

	// signal.NotifyContext gives us a context that's automatically
	// cancelled when the OS sends SIGINT (Ctrl+C) or SIGTERM (what
	// Docker/systemd send when stopping a service). This is the standard
	// Go pattern for graceful shutdown — everything downstream that
	// accepts a context (like our DB calls) can react to this same signal.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run the server in a background goroutine so main() can keep going
	// and block on waiting for the shutdown signal instead.
	go func() {
		log.Printf("🚀 server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Block here until Ctrl+C / SIGTERM arrives.
	<-ctx.Done()
	log.Println("🛑 shutdown signal received, draining in-flight requests...")

	// Give in-flight requests up to 10 seconds to finish before forcing
	// shutdown — this prevents cutting off a request mid-DB-write.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ forced shutdown: %v", err)
	} else {
		log.Println("✅ server shut down cleanly")
	}
}

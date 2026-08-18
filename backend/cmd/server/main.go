package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fpl-assistant/internal/actions"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

func main() {
	portFlag := flag.Int("port", 18492, "Port to listen on (default: 18492)")
	flag.Parse()

	cache := fpl.NewMemoryCache()
	client := fpl.NewHTTPClient(cache)
	engine := scoring.NewXPEngine()
	optimizer := scoring.NewSquadOptimizer()

	router := actions.NewRouter(client, engine, optimizer)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *portFlag))
	if err != nil {
		log.Fatalf("[FATAL] Failed to bind port %d: %v", *portFlag, err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	log.Printf("[FPL ASSISTANT CORE] Server listening on http://127.0.0.1:%d", actualPort)

	srv := &http.Server{
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	// Signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[FPL ASSISTANT CORE] Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[ERROR] Server forced shutdown: %v", err)
	}
	log.Println("[FPL ASSISTANT CORE] Server stopped.")
}

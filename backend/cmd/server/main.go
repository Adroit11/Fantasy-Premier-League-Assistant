package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fpl-assistant/internal/actions"
	"fpl-assistant/internal/config"
	"fpl-assistant/internal/db"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

func main() {
	cfg := config.LoadConfig()

	portFlag := flag.Int("port", cfg.Port, "Port to listen on")
	flag.Parse()

	database, err := db.Connect(cfg)
	if err != nil {
		log.Printf("[MySQL] Init error: %v (continuing with in-memory cache)", err)
	}

	var cache fpl.Cache
	var hybrid *db.HybridCache
	if database != nil {
		hybrid = db.NewHybridCache(database)
		cache = hybrid
		log.Println("[CACHE] Hybrid L1 memory + L2 MySQL cache enabled")
	} else {
		cache = fpl.NewMemoryCache()
		log.Println("[CACHE] In-memory cache only")
	}

	client := fpl.NewHTTPClient(cache, cfg.FPLBaseURL, cfg.FPLUserAgent)
	engine := scoring.NewXPEngine()
	optimizer := scoring.NewSquadOptimizer()
	router := actions.NewRouter(client, engine, optimizer, database)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *portFlag))
	if err != nil {
		log.Fatalf("[FATAL] Failed to bind port %d: %v", *portFlag, err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	log.Printf("[FPL ASSISTANT CORE] Environment: %s", cfg.Environment)
	log.Printf("[FPL ASSISTANT CORE] FPL API Base URL: %s", cfg.FPLBaseURL)
	log.Printf("[FPL ASSISTANT CORE] Server listening on http://127.0.0.1:%d", actualPort)

	go func() {
		if err := router.App.Listener(listener); err != nil {
			log.Printf("[ERROR] HTTP server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[FPL ASSISTANT CORE] Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := router.App.ShutdownWithContext(ctx); err != nil {
		log.Printf("[ERROR] Server forced shutdown: %v", err)
	}

	if hybrid != nil {
		hybrid.Close()
	} else if mem, ok := cache.(*fpl.MemoryCache); ok {
		mem.Close()
	}
	if err := database.Close(); err != nil {
		log.Printf("[ERROR] Closing MySQL: %v", err)
	}

	log.Println("[FPL ASSISTANT CORE] Server stopped.")
}

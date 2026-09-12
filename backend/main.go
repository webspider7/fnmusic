package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fn-lx-player/pkg/api"
	"fn-lx-player/pkg/config"
	"fn-lx-player/pkg/sources"
)

//go:embed all:dist
var frontendDist embed.FS

func main() {
	port := flag.Int("port", 8899, "HTTP server listening port")
	dataDir := flag.String("data", "./data", "Data directory for persistent configuration")
	flag.Parse()

	log.Printf("[INIT] Starting Aurora Music (fn-lx-player) on port %d, data directory: %s", *port, *dataDir)

	// 1. Prepare frontend static FS
	var staticFS fs.FS
	sub, err := fs.Sub(frontendDist, "dist")
	if err == nil {
		staticFS = sub
	}

	// 2. Initialize Config & Sources Manager (Pure Custom Sources)
	cfgMgr, err := config.NewConfigManager(*dataDir, *port)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize config manager: %v", err)
	}

	sourcesMgr := sources.NewManager(cfgMgr, nil, "")

	// 4. Initialize HTTP API Server
	server := api.NewServer(cfgMgr, sourcesMgr, staticFS)

	httpServer := &http.Server{
		Handler:      server.Router(),
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		// Fallback to IPv4 if dual-stack failed
		listener, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
		if err != nil {
			log.Fatalf("[FATAL] Failed to bind to port %d: %v", *port, err)
		}
	}

	go func() {
		log.Printf("[READY] Server running and listening on http://0.0.0.0:%d", *port)
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[SHUTDOWN] Stopping Aurora Music...")
}

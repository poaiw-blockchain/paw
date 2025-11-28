package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paw-chain/paw/explorer/indexer/config"
	"github.com/paw-chain/paw/explorer/indexer/internal/api"
	"github.com/paw-chain/paw/explorer/indexer/internal/cache"
	"github.com/paw-chain/paw/explorer/indexer/internal/database"
	"github.com/paw-chain/paw/explorer/indexer/internal/indexer"
	"github.com/paw-chain/paw/explorer/indexer/internal/metrics"
	"github.com/paw-chain/paw/explorer/indexer/internal/subscriber"
	"github.com/paw-chain/paw/explorer/indexer/internal/websocket"
	"github.com/paw-chain/paw/explorer/indexer/pkg/logger"
)

var (
	configPath = flag.String("config", "config/config.yaml", "path to configuration file")
	version    = "1.0.0"
	buildTime  = "unknown"
)

func main() {
	flag.Parse()

	// Initialize logger
	log := logger.NewLogger("indexer")
	log.Info("Starting PAW Chain Explorer Indexer", "version", version, "build_time", buildTime)

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize metrics
	metricsServer := metrics.NewServer(cfg.Metrics.Port)
	go func() {
		if err := metricsServer.Start(); err != nil {
			log.Error("Metrics server failed", "error", err)
		}
	}()

	// Initialize database
	log.Info("Connecting to database", "host", cfg.Database.Host, "port", cfg.Database.Port)
	db, err := database.NewDatabase(cfg.Database)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	log.Info("Running database migrations")
	if err := db.RunMigrations(); err != nil {
		log.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize Redis cache
	log.Info("Connecting to Redis", "host", cfg.Redis.Host, "port", cfg.Redis.Port)
	redisCache, err := cache.NewRedisCache(cfg.Redis)
	if err != nil {
		log.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisCache.Close()

	// Initialize blockchain subscriber
	log.Info("Initializing blockchain subscriber", "rpc_url", cfg.Chain.RPCURL)
	sub, err := subscriber.NewSubscriber(cfg.Chain, db, redisCache, log)
	if err != nil {
		log.Error("Failed to initialize subscriber", "error", err)
		os.Exit(1)
	}

	// Initialize indexer
	log.Info("Initializing blockchain indexer")
	idx := indexer.NewIndexer(db, redisCache, sub, cfg.Indexer, log)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(log)
	go wsHub.Run()

	// Initialize API server
	log.Info("Initializing API server", "port", cfg.API.Port)
	apiServer := api.NewServer(cfg.API, db, redisCache, wsHub, log)

	// Start indexer
	log.Info("Starting blockchain indexer")
	go func() {
		if err := idx.Start(ctx); err != nil {
			log.Error("Indexer failed", "error", err)
			cancel()
		}
	}()

	// Start API server
	log.Info("Starting API server")
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Error("API server failed", "error", err)
			cancel()
		}
	}()

	// Subscribe to new blocks
	go func() {
		if err := sub.SubscribeToBlocks(ctx, wsHub); err != nil {
			log.Error("Block subscription failed", "error", err)
			cancel()
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Info("Received interrupt signal, shutting down gracefully")
	case <-ctx.Done():
		log.Info("Context cancelled, shutting down")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	log.Info("Stopping indexer")
	idx.Stop()

	log.Info("Stopping API server")
	if err := apiServer.Stop(shutdownCtx); err != nil {
		log.Error("Failed to stop API server gracefully", "error", err)
	}

	log.Info("Stopping WebSocket hub")
	wsHub.Stop()

	log.Info("Stopping metrics server")
	if err := metricsServer.Stop(shutdownCtx); err != nil {
		log.Error("Failed to stop metrics server gracefully", "error", err)
	}

	log.Info("PAW Chain Explorer Indexer stopped successfully")
}

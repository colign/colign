package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gobenpark/colign/internal/config"
	"github.com/gobenpark/colign/internal/database"
	"github.com/gobenpark/colign/internal/server"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "Run migrations and exit")
	skipMigrate := flag.Bool("skip-migrate", false, "Skip auto-migration on startup")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Run migrations
	migrationsPath := cfg.MigrationsPath
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}
	if !*skipMigrate {
		if err := database.RunMigrations(cfg.DatabaseURL, migrationsPath); err != nil {
			if *migrateOnly {
				log.Fatalf("migration failed: %v", err)
			}
			log.Printf("warning: migration failed: %v", err)
		}
	}

	if *migrateOnly {
		log.Println("migrations completed successfully")
		return
	}

	s, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	defer func() { _ = s.Close() }()

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      s.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server starting on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server exited")
}

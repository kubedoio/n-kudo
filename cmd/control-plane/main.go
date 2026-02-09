package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	controlplane "github.com/kubedoio/n-kudo/internal/controlplane/api"
	store "github.com/kubedoio/n-kudo/internal/controlplane/db"
	"github.com/kubedoio/n-kudo/internal/controlplane/db/migrate"
	_ "github.com/lib/pq"
)

func main() {
	cfg := controlplane.LoadConfig()
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			if err := runMigrate(cfg); err != nil {
				log.Fatal(err)
			}
			return
		case "serve":
			if err := runServe(cfg); err != nil {
				log.Fatal(err)
			}
			return
		}
	}
	if err := runServe(cfg); err != nil {
		log.Fatal(err)
	}
}

func runMigrate(cfg controlplane.Config) error {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	dir := fs.String("dir", "db/migrations", "migrations directory")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	absDir := *dir
	if !filepath.IsAbs(absDir) {
		cwd, _ := os.Getwd()
		absDir = filepath.Join(cwd, absDir)
	}
	if err := migrate.Up(ctx, db, absDir); err != nil {
		return err
	}
	fmt.Println("migrations applied")
	return nil
}

func runServe(cfg controlplane.Config) error {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// Configure connection pool settings
	store.ConfigureConnectionPool(db)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := db.PingContext(startCtx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	repo := store.NewPostgresRepo(db)
	app, err := controlplane.NewApp(cfg, repo)
	if err != nil {
		return err
	}
	app.StartBackgroundWorkers(ctx)
	tlsCfg, err := app.TLSConfig()
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      app.Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		TLSConfig:    tlsCfg,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("control-plane listening on https://%s", cfg.ListenAddr)
		errCh <- server.ListenAndServeTLS("", "")
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received, starting graceful shutdown...")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	// Create shutdown context with 30 second timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown the HTTP server gracefully
	log.Println("shutting down HTTP server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server shutdown complete")
	}

	// Close the database repository
	log.Println("closing database connection...")
	if err := repo.Close(); err != nil {
		log.Printf("database close error: %v", err)
	} else {
		log.Println("database connection closed")
	}

	log.Println("graceful shutdown complete")
	return nil
}

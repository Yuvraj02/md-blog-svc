package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marketing-digest/pkg/logger"

	"github.com/marketing-digest/blog-service/internal/app/article"
	"github.com/marketing-digest/blog-service/internal/app/blog"
	"github.com/marketing-digest/blog-service/internal/config"
	"github.com/marketing-digest/blog-service/internal/infrastructure/database"
	"github.com/marketing-digest/blog-service/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "blog fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	log := logger.New(cfg.Service)
	log.Info("starting", "env", cfg.AppEnv, "grpc_port", cfg.GRPCPort)

	db, err := database.Open(cfg)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer func() {
		if cerr := database.Close(db); cerr != nil {
			log.Error("database close", "error", cerr)
		}
	}()

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.Ping(pingCtx, db); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	blogStore := blog.NewGORMStore(db)
	articleStore := article.NewGORMStore(db)
	blogSvc := blog.NewService(blogStore)
	articleSvc := article.NewService(articleStore, blogStore)

	srv := server.New(blogSvc, articleSvc, log, cfg.GRPCPort)
	srv.SetReady(true)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		log.Info("shutdown signal", "signal", sig.String())
	}

	srv.SetReady(false)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	srv.GracefulStop(shutdownCtx)
	log.Info("shutdown complete")
	return nil
}

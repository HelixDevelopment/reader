package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/helixreader/helix/server/internal/aimemory"
	"github.com/helixreader/helix/server/internal/catalog"
	"github.com/helixreader/helix/server/internal/config"
	"github.com/helixreader/helix/server/internal/httpapi"
	"github.com/helixreader/helix/server/internal/identify"
	"github.com/helixreader/helix/server/internal/ingest"
	"github.com/helixreader/helix/server/internal/kosync"
	"github.com/helixreader/helix/server/internal/sources"
	"github.com/helixreader/helix/server/internal/syncstore"
	"github.com/quic-go/quic-go/http3"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	cfg, err := config.FromEnv()
	if err != nil {
		slog.Error("configuration", "error", err)
		os.Exit(2)
	}
	if cfg.OPDSUsername == "" {
		slog.Warn("read API and OPDS authentication are disabled; configure HELIX_OPDS_USERNAME and HELIX_OPDS_PASSWORD before non-local exposure")
	}
	var repo catalog.Repository
	if cfg.CatalogDSN != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		postgres, err := catalog.NewPostgres(ctx, cfg.CatalogDSN)
		cancel()
		if err != nil {
			slog.Error("PostgreSQL initialization", "error", err)
			os.Exit(2)
		}
		repo = postgres
	} else {
		repo = catalog.NewMemory()
		slog.Warn("using volatile in-memory catalog; configure HELIX_POSTGRES_DSN for durable storage")
	}
	defer repo.Close()
	api := httpapi.New(cfg, repo, sources.DefaultRegistry(), &ingest.Service{Repository: repo, Identifier: &identify.Pipeline{}})
	defer api.Close()
	if cfg.AIMemory {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		memory, err := aimemory.NewPostgres(ctx, cfg.CatalogDSN, cfg.EmbeddingDimensions)
		cancel()
		if err != nil {
			slog.Error("AI memory initialization", "error", err)
			os.Exit(2)
		}
		api.Memory = memory
		defer api.Memory.Close()
	}
	if cfg.RedisURL != "" {
		options, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			slog.Error("Redis configuration", "error", err)
			os.Exit(2)
		}
		client := redis.NewClient(options)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = client.Ping(ctx).Err()
		cancel()
		if err != nil {
			client.Close()
			slog.Error("Redis initialization", "error", err)
			os.Exit(2)
		}
		api.Progress = syncstore.NewRedis(client)
		defer api.Progress.Close()

		// KOReader sync owns a separate client so each store has one clear
		// connection lifecycle. Registration is opt-in; all verifiers and
		// progress records use the same durable Redis deployment.
		kosyncClient := redis.NewClient(options)
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		err = kosyncClient.Ping(ctx).Err()
		cancel()
		if err != nil {
			kosyncClient.Close()
			slog.Error("KOReader sync Redis initialization", "error", err)
			os.Exit(2)
		}
		api.KOSync = kosync.NewService(kosync.NewRedis(kosyncClient), cfg.KOSyncRegistration)
		defer api.KOSync.Close()
	}
	if err := run(cfg, api); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, api *httpapi.Server) error {
	if gin.Mode() == gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	handler := api.Router()
	tcp := &http.Server{Addr: cfg.Address, Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 0, IdleTimeout: 120 * time.Second, MaxHeaderBytes: 1 << 20, ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn)}
	var quicServer *http3.Server
	if cfg.TLSCertFile != "" {
		tcp.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, NextProtos: []string{"h2", "http/1.1"}}
	}
	if cfg.TLSCertFile != "" && cfg.HTTP3 {
		quicServer = &http3.Server{Addr: cfg.Address, Handler: handler, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS13}, MaxHeaderBytes: 1 << 20, IdleTimeout: 120 * time.Second}
	}
	errCh := make(chan error, 2)
	go func() {
		slog.Info("TCP listener starting", "address", cfg.Address, "tls", cfg.TLSCertFile != "")
		var err error
		if cfg.TLSCertFile != "" {
			err = tcp.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			err = tcp.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("TCP: %w", err)
		} else {
			errCh <- nil
		}
	}()
	if quicServer != nil {
		go func() {
			slog.Info("HTTP/3 listener starting", "address", cfg.Address)
			if err := quicServer.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("HTTP/3: %w", err)
			} else {
				errCh <- nil
			}
		}()
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case sig := <-signals:
		slog.Info("shutdown requested", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	var errs []error
	if quicServer != nil {
		if err := quicServer.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if err := tcp.Shutdown(ctx); err != nil {
		errs = append(errs, err)
	}
	// HTTP draining alone does not cover background ingestion. Stop accepting
	// scans, cancel them, and drain workers before deferred connector cleanup.
	if err := api.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("scan shutdown: %w", err))
	}
	return errors.Join(errs...)
}

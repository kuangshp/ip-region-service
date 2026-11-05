package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/box-im/ip-region-service/initialize"
	"github.com/box-im/ip-region-service/pkg/ipregion"
	"github.com/gin-gonic/gin"
)

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	client, err := ipregion.New(ipregion.Config{
		IPv4Path: env("IP2REGION_V4_PATH", "data/ip2region_v4.xdb"),
		IPv6Path: env("IP2REGION_V6_PATH", "data/ip2region_v6.xdb"),
	})
	if err != nil {
		return err
	}
	defer client.Close()
	gin.SetMode(gin.ReleaseMode)
	server := &http.Server{Addr: env("HTTP_ADDR", ":8080"), Handler: initialize.Router(client),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	slog.Info("IP region service starting", "address", server.Addr)
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return err
		}
		return nil
	}
}

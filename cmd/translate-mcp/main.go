package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/mohammadraufzahed/translate-mcp/internal/config"
	"github.com/mohammadraufzahed/translate-mcp/internal/server"
)

func main() {
	var (
		configPath = flag.String("config", "", "Path to config file")
		transport  = flag.String("transport", "", "Transport: stdio or http")
		port       = flag.Int("port", 0, "HTTP port")
		host       = flag.String("host", "", "HTTP host")
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	if *transport != "" {
		cfg.Server.Transport = *transport
	}
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}

	setupLogging(cfg)

	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
	defer srv.Close()

	if err := srv.Run(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func setupLogging(cfg *config.Config) {
	level := slog.LevelInfo
	switch cfg.Logging.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cfg.Logging.Format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(handler))
}

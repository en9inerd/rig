package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/en9inerd/rig/internal/config"
	"github.com/en9inerd/rig/internal/log"
	"github.com/en9inerd/rig/internal/notify"
	"github.com/en9inerd/rig/internal/runtime"
	"github.com/en9inerd/rig/internal/storage"
	"github.com/en9inerd/rig/internal/tasks/feedwatch"
	"github.com/en9inerd/rig/internal/tasks/ipwatch"
	"github.com/en9inerd/rig/internal/tasks/visitor"
)

var version = "dev"

func versionString() string {
	var revision, buildTime string
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.revision":
				if len(kv.Value) >= 7 {
					revision = kv.Value[:7]
				}
			case "vcs.time":
				buildTime = kv.Value
			}
		}
	}
	s := "rig version " + version
	if revision != "" {
		s += " (" + revision + ")"
	}
	if buildTime != "" {
		s += " built " + buildTime
	}
	return s
}

func run(ctx context.Context, args []string, getenv func(string) string) error {
	for _, a := range args[1:] {
		if a == "--version" || a == "-version" {
			fmt.Println(versionString())
			return nil
		}
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.ParseConfig(args, getenv)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	logger := log.NewLogger(cfg.Verbose)
	logger.Info("starting rig",
		"version", version,
		"addr", cfg.HTTPAddr,
		"store", cfg.StorePath,
		"visitor", cfg.Visitor.Enabled,
		"feed", cfg.Feed.Enabled,
		"ip", cfg.IP.Enabled,
	)

	store, err := storage.New(cfg.StorePath)
	if err != nil {
		return fmt.Errorf("failed to open store: %w", err)
	}

	notifier := notify.NewTelegram(cfg.TelegramBotToken)
	rt := runtime.New(logger, cfg.HTTPAddr, cfg.CORSOrigin)

	if cfg.Visitor.Enabled {
		rt.Register(visitor.New(notifier, logger, cfg.Visitor))
	}
	if cfg.Feed.Enabled {
		rt.Register(feedwatch.New(notifier, logger, cfg.Feed, store))
	}
	if cfg.IP.Enabled {
		rt.Register(ipwatch.New(notifier, logger, cfg.IP, store))
	}

	return rt.Run(ctx)
}

func main() {
	if err := run(context.Background(), os.Args, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

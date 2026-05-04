package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/en9inerd/rig/internal/config"
	"github.com/en9inerd/rig/internal/healthcheck"
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

func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func initVisitorSites() error {
	token, err := generateToken()
	if err != nil {
		return err
	}
	sites := []visitor.Site{
		{Name: "site", AuthToken: token, ChatID: "CHANGE_ME", Tag: "site"},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(sites)
}

func run(ctx context.Context, args []string, getenv func(string) string) error {
	cfg := config.ParseConfig(getenv)

	for _, a := range args[1:] {
		switch a {
		case "--version", "-version":
			fmt.Println(versionString())
			return nil
		case "--init":
			return initVisitorSites()
		case "--healthcheck":
			addr := cfg.HTTPAddr

			if err := healthcheck.Check(addr, cfg.TLS.Enabled()); err != nil {
				return err
			}
			return nil
		}
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger := log.NewLogger(cfg.Verbose)

	vcfg, err := visitor.LoadConfig(getenv)
	if err != nil {
		return fmt.Errorf("visitor config: %w", err)
	}
	fcfg, err := feedwatch.LoadConfig(getenv)
	if err != nil {
		return fmt.Errorf("feedwatch config: %w", err)
	}
	icfg, err := ipwatch.LoadConfig(getenv)
	if err != nil {
		return fmt.Errorf("ipwatch config: %w", err)
	}

	if (vcfg != nil || fcfg != nil || icfg != nil) && cfg.TelegramBotToken == "" {
		return fmt.Errorf("RIG_TELEGRAM_BOT_TOKEN is required when tasks are enabled")
	}

	store, err := storage.New(cfg.StorePath)
	if err != nil {
		return fmt.Errorf("failed to open store: %w", err)
	}

	notifier := notify.NewTelegram(cfg.TelegramBotToken)
	rt := runtime.New(logger, cfg.HTTPAddr, cfg.CORSOrigin, cfg.TLS)

	if vcfg != nil {
		rt.Register(visitor.New(notifier, logger, *vcfg))
	}
	if fcfg != nil {
		rt.Register(feedwatch.New(notifier, logger, *fcfg, store))
	}
	if icfg != nil {
		rt.Register(ipwatch.New(notifier, logger, *icfg, store))
	}

	logger.Info("starting rig",
		"version", version,
		"addr", cfg.HTTPAddr,
		"store", cfg.StorePath,
		"visitor", vcfg != nil,
		"feed", fcfg != nil,
		"ip", icfg != nil,
	)

	return rt.Run(ctx)
}

func main() {
	if err := run(context.Background(), os.Args, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

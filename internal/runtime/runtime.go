package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/en9inerd/go-pkgs/middleware"
	"github.com/en9inerd/go-pkgs/router"
	"github.com/en9inerd/rig/internal/config"
	"github.com/en9inerd/rig/internal/tasks"
)

type Runtime struct {
	tasks  []tasks.Task
	logger *slog.Logger
	addr   string
	tls    config.TLSConfig
	router *router.Group
}

func New(logger *slog.Logger, addr, corsOrigin string, tls config.TLSConfig) *Runtime {
	r := router.New(http.NewServeMux())

	r.Use(
		middleware.CORS(middleware.CORSConfig{
			Origin:  corsOrigin,
			Methods: []string{"POST"},
			Headers: []string{"Content-Type"},
		}),
		middleware.Headers("X-Content-Type-Options: nosniff"),
		middleware.RealIP,
		middleware.Recoverer(logger, false),
		middleware.GlobalThrottle(1000),
		middleware.Health,
		middleware.SizeLimit(64<<10),
		middleware.Logger(logger),
	)

	return &Runtime{
		logger: logger,
		addr:   addr,
		tls:    tls,
		router: r,
	}
}

func (rt *Runtime) Register(t tasks.Task) {
	rt.tasks = append(rt.tasks, t)
}

func (rt *Runtime) Run(ctx context.Context) error {
	for _, t := range rt.tasks {
		if ht, ok := t.(tasks.HTTPTask); ok {
			group := rt.router.Group()
			ht.RegisterRoutes(group)
			rt.logger.Info("registered HTTP routes", "task", t.Name())
		}
	}

	httpServer := &http.Server{
		Addr:              rt.addr,
		Handler:           rt.router,
		ErrorLog:          slog.NewLogLogger(rt.logger.Handler(), slog.LevelWarn),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	var wg sync.WaitGroup

	wg.Go(func() {
		<-ctx.Done()
		rt.logger.Info("shutting down HTTP server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			rt.logger.Error("HTTP server shutdown error", "error", err)
		}

		// HTTP server fully stopped — all handlers have returned.
		// Drain any background work spawned by handlers.
		for _, t := range rt.tasks {
			if s, ok := t.(tasks.Stopper); ok {
				s.Stop()
			}
		}
	})

	for _, t := range rt.tasks {
		wg.Go(func() {
			rt.logger.Info("starting task", "task", t.Name())
			if err := t.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
				rt.logger.Error("task failed", "task", t.Name(), "error", err)
			}
			rt.logger.Info("task stopped", "task", t.Name())
		})
	}

	var err error
	if rt.tls.Enabled() {
		rt.logger.Info("listening (TLS)", "addr", rt.addr)
		err = httpServer.ListenAndServeTLS(rt.tls.CertFile, rt.tls.KeyFile)
	} else {
		rt.logger.Info("listening", "addr", rt.addr)
		err = httpServer.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server error: %w", err)
	}

	wg.Wait()
	return nil
}

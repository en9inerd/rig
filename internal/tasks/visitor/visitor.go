package visitor

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time" // also used by commented-out deduplication logic

	"github.com/en9inerd/go-pkgs/httpclient"
	"github.com/en9inerd/go-pkgs/httpjson"
	"github.com/en9inerd/go-pkgs/realip"
	"github.com/en9inerd/go-pkgs/router"
	"github.com/en9inerd/rig/internal/config"
	"github.com/en9inerd/rig/internal/notify"
	"github.com/oschwald/geoip2-golang"
)

// Uncomment to enable deduplication (one notification per IP+URL per window).
// const dedupeWindow = 10 * time.Minute

type Task struct {
	notifier notify.Notifier
	logger   *slog.Logger
	cfg      config.VisitorConfig
	client   *httpclient.Client
	geodb    *geoip2.Reader
	inflight sync.WaitGroup
	// seen  sync.Map // IP+URL → time.Time (used with deduplication)
}

func New(notifier notify.Notifier, logger *slog.Logger, cfg config.VisitorConfig) *Task {
	t := &Task{
		notifier: notifier,
		logger:   logger.With("task", "visitor"),
		cfg:      cfg,
		client: httpclient.NewWithConfig(httpclient.Config{
			Timeout: 10 * time.Second,
		}),
	}

	if cfg.GeoIPDB != "" {
		db, err := geoip2.Open(cfg.GeoIPDB)
		if err != nil {
			t.logger.Warn("GeoIP database not available, geolocation disabled", "path", cfg.GeoIPDB, "error", err)
		} else {
			t.geodb = db
			t.logger.Info("loaded GeoIP database", "path", cfg.GeoIPDB)
		}
	}

	return t
}

func (t *Task) Name() string { return "visitor" }

func (t *Task) Start(ctx context.Context) error {
	// Uncomment to enable periodic cleanup of the deduplication map.
	// ticker := time.NewTicker(dedupeWindow)
	// defer ticker.Stop()
	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		return ctx.Err()
	// 	case <-ticker.C:
	// 		cutoff := time.Now().Add(-dedupeWindow)
	// 		t.seen.Range(func(key, value any) bool {
	// 			if value.(time.Time).Before(cutoff) {
	// 				t.seen.Delete(key)
	// 			}
	// 			return true
	// 		})
	// 	}
	// }

	<-ctx.Done()
	return ctx.Err()
}

func (t *Task) Stop() {
	t.inflight.Wait()
	if t.geodb != nil {
		t.geodb.Close()
	}
}

func (t *Task) RegisterRoutes(g *router.Group) {
	g.HandleFunc("POST /{token}/visitor-notifier", t.handleVisitor)
}

type visitorPayload struct {
	URL              string `json:"url"`
	ScreenDimensions string `json:"screenDimensions"`
	Referrer         string `json:"referrer"`
	UserAgent        string `json:"userAgent"`
	HasTouchScreen   bool   `json:"hasTouchScreen"`
}

type ipifyResponse struct {
	IP string `json:"ip"`
}

func (t *Task) handleVisitor(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("token") != t.cfg.AuthToken {
		http.NotFound(w, r)
		return
	}

	var payload visitorPayload
	if err := httpjson.DecodeJSONWithLimit(r, &payload, 64<<10); err != nil {
		t.logger.Error("failed to decode request", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Extract IP before returning the response — *http.Request must not be
	// read after the handler returns because the server may recycle it.
	clientIP, err := realip.Get(r)
	if err != nil {
		t.logger.Error("failed to get client IP", "error", err)
		clientIP = "unknown"
	}

	w.WriteHeader(http.StatusOK)

	// Uncomment to enable deduplication (one notification per IP+URL per window).
	// dedupeKey := clientIP + "|" + payload.URL
	// if v, ok := t.seen.Load(dedupeKey); ok {
	// 	if time.Since(v.(time.Time)) < dedupeWindow {
	// 		return
	// 	}
	// }
	// t.seen.Store(dedupeKey, time.Now())

	t.inflight.Go(func() {
		t.processVisitor(clientIP, payload)
	})
}

func (t *Task) processVisitor(clientIP string, payload visitorPayload) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// When the extracted IP is private (e.g. Docker bridge, LAN), resolve the
	// server's public IP via ipify so geolocation still works.
	displayIP := clientIP
	ip := net.ParseIP(clientIP)
	if ip != nil && realip.IsPrivateIP(ip) {
		var result ipifyResponse
		if err := t.client.GetJSON(ctx, "https://api64.ipify.org?format=json", &result); err == nil {
			displayIP = result.IP
		} else {
			t.logger.Error("ipify fallback failed", "error", err)
		}
	}

	location := t.geolocate(displayIP)

	timestamp := time.Now().Format("January 2, 2006 3:04 PM")

	lines := []string{
		fmt.Sprintf("Tag: %s", t.cfg.Tag),
		fmt.Sprintf("When: %s", timestamp),
		fmt.Sprintf("IP: %s", displayIP),
		fmt.Sprintf("Location: %s", location),
		fmt.Sprintf("URL: %s", payload.URL),
		fmt.Sprintf("Screen Dimensions: %s", payload.ScreenDimensions),
		fmt.Sprintf("Has Touchscreen: %t", payload.HasTouchScreen),
		fmt.Sprintf("Referrer: %s", payload.Referrer),
		fmt.Sprintf("User Agent: %s", payload.UserAgent),
	}

	if err := t.notifier.Send(ctx, notify.Message{
		ChatID:  t.cfg.ChatID,
		Content: strings.Join(lines, "\n"),
		Options: notify.MessageOptions{
			DisableWebPagePreview: true,
		},
	}); err != nil {
		t.logger.Error("failed to send notification", "error", err)
	}
}

func (t *Task) geolocate(ipStr string) string {
	if t.geodb == nil || ipStr == "" || ipStr == "unknown" {
		return ""
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	record, err := t.geodb.City(ip)
	if err != nil {
		t.logger.Error("geolocation lookup failed", "ip", ipStr, "error", err)
		return ""
	}

	var parts []string
	if city := record.City.Names["en"]; city != "" {
		parts = append(parts, city)
	}
	if len(record.Subdivisions) > 0 {
		if region := record.Subdivisions[0].Names["en"]; region != "" {
			parts = append(parts, region)
		}
	}
	if country := record.Country.Names["en"]; country != "" {
		parts = append(parts, country)
	}

	return strings.Join(parts, ", ")
}

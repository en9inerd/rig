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
	"github.com/en9inerd/rig/internal/notify"
	"github.com/oschwald/geoip2-golang"
)

// Uncomment to enable deduplication (one notification per IP+URL per window).
// const dedupeWindow = 10 * time.Minute

type Task struct {
	notifier notify.Notifier
	logger   *slog.Logger
	sites    map[string]*Site // authToken → site
	client   *httpclient.Client
	geodb    *geoip2.Reader
	inflight sync.WaitGroup
	// seen  sync.Map // IP+URL → time.Time (used with deduplication)
}

func New(notifier notify.Notifier, logger *slog.Logger, cfg Config) *Task {
	sites := make(map[string]*Site, len(cfg.Sites))
	for i := range cfg.Sites {
		sites[cfg.Sites[i].AuthToken] = &cfg.Sites[i]
	}

	t := &Task{
		notifier: notifier,
		logger:   logger.With("task", "visitor"),
		sites:    sites,
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
	g.HandleFunc("POST /{token}/visitor", t.handleVisitor)
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
	site, ok := t.sites[r.PathValue("token")]
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Replace the token in the path so the Logger middleware doesn't log it.
	r.URL.Path = "/" + site.Name + "/visitor"

	var payload visitorPayload
	if err := httpjson.DecodeJSONWithLimit(r, &payload, 64<<10); err != nil {
		t.logger.Error("failed to decode request", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Extract headers before returning the response — *http.Request must
	// not be read after the handler returns because the server may recycle it.
	clientIP, err := realip.Get(r)
	if err != nil {
		t.logger.Error("failed to get client IP", "error", err)
		clientIP = "unknown"
	}
	lang := primaryLanguage(r.Header.Get("Accept-Language"))

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
		t.processVisitor(site, clientIP, lang, payload)
	})
}

func (t *Task) processVisitor(site *Site, clientIP, lang string, payload visitorPayload) {
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
		fmt.Sprintf("Tag: %s", site.Tag),
		fmt.Sprintf("When: %s", timestamp),
		fmt.Sprintf("IP: %s", displayIP),
		fmt.Sprintf("Location: %s", location),
		fmt.Sprintf("URL: %s", payload.URL),
		fmt.Sprintf("Screen Dimensions: %s", payload.ScreenDimensions),
		fmt.Sprintf("Has Touchscreen: %t", payload.HasTouchScreen),
		fmt.Sprintf("Referrer: %s", payload.Referrer),
		fmt.Sprintf("User Agent: %s", payload.UserAgent),
		fmt.Sprintf("Language: %s", lang),
	}

	if err := t.notifier.Send(ctx, notify.Message{
		ChatID:  site.ChatID,
		Content: strings.Join(lines, "\n"),
		Options: notify.MessageOptions{
			DisableWebPagePreview: true,
		},
	}); err != nil {
		t.logger.Error("failed to send notification", "error", err)
	}
}

// primaryLanguage extracts the first language tag from an Accept-Language header
// value, e.g. "en-US" from "en-US,en;q=0.9,de;q=0.8".
func primaryLanguage(accept string) string {
	if accept == "" {
		return ""
	}
	tag, _, _ := strings.Cut(accept, ",")
	tag, _, _ = strings.Cut(tag, ";")
	return strings.TrimSpace(tag)
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

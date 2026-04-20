# Rig

A lightweight automation runtime in Go. Each unit of work is a **task** - a self-contained module that wires an event source to one or more actions. The runtime manages task lifecycle, shared services, and the HTTP server.

## Tasks

| Task | Type | Description |
|---|---|---|
| **Visitor Notifier** | HTTP endpoint | Receives visitor data via `POST /{token}/visitor`, geolocates the IP using a local MaxMind database, and sends a Telegram notification. Supports multiple sites, each with its own token and tag. |
| **Feed Watcher** | Ticker | Polls an Atom feed at a configurable interval and sends new posts to Telegram |
| **IP Watcher** | Ticker | Monitors the server's public IP and sends a Telegram notification on change |

Tasks are independent, each manages its own goroutine(s), state, and config. Disable any task via environment variables.

## Quick Start

```bash
cp .env.example .env
# Edit .env with your Telegram bot token, chat IDs, and MaxMind credentials

# Generate visitors.json with a random auth token (pick one)
rig --init > visitors.json                           # local binary
docker run --rm enginerd/rig --init > visitors.json  # from Docker image
# Edit visitors.json with your site names and chat IDs

docker compose up -d
```

The `geoipupdate` sidecar downloads the GeoLite2-City database before `rig` starts. A free [MaxMind account](https://www.maxmind.com/en/geolite2/signup) is required.

## Configuration

All configuration is via environment variables.

### Global

| Variable | Description | Default |
|---|---|---|
| `RIG_HTTP_ADDR` | HTTP listen address | `:8080` |
| `RIG_TELEGRAM_BOT_TOKEN` | Telegram Bot API token | - |
| `RIG_STORE_PATH` | Path to persistent state file | `/data/rig.json` |
| `RIG_CORS_ORIGIN` | Allowed CORS origin | - |
| `RIG_TLS_CERT` | Path to TLS certificate file | - |
| `RIG_TLS_KEY` | Path to TLS private key file | - |
| `RIG_VERBOSE` | Enable debug logging | `false` |

### Visitor Notifier

| Variable | Description | Default |
|---|---|---|
| `RIG_VISITOR_ENABLED` | Enable task | `true` |
| `RIG_VISITOR_SITES_FILE` | Path to JSON file defining visitor sites | - |
| `RIG_VISITOR_GEOIP_DB` | Path to MaxMind GeoLite2-City database | `/data/geoip/GeoLite2-City.mmdb` |
| `RIG_VISITOR_DEDUP` | Drop repeat notifications for same IP+URL within the window | `false` |
| `RIG_VISITOR_DEDUP_WINDOW` | Deduplication window (when enabled) | `10m` |

The sites file is a JSON array. Generate a starter file with `rig --init`, or create one manually:

```json
[
  {"name": "blog", "authToken": "TOKEN", "chatId": "CHAT_ID", "tag": "Blog"}
]
```

Each site gets its own endpoint: `POST /{authToken}/visitor`. The `tag` field is optional and defaults to the site name.

### Feed Watcher

| Variable | Description | Default |
|---|---|---|
| `RIG_FEED_ENABLED` | Enable task | `true` |
| `RIG_FEED_URL` | Atom/RSS feed URL | - |
| `RIG_FEED_INTERVAL` | Poll interval | `15m` |
| `RIG_FEED_CHAT_ID` | Telegram chat ID | - |

### IP Watcher

| Variable | Description | Default |
|---|---|---|
| `RIG_IP_ENABLED` | Enable task | `true` |
| `RIG_IP_INTERVAL` | Poll interval | `15m` |
| `RIG_IP_CHAT_ID` | Telegram chat ID | - |

### MaxMind GeoIP (geoipupdate sidecar)

| Variable | Description |
|---|---|
| `GEOIPUPDATE_ACCOUNT_ID` | MaxMind account ID |
| `GEOIPUPDATE_LICENSE_KEY` | MaxMind license key |

## Development

```bash
# Build
make build

# Run locally (loads .env)
make run

# Run with debug logging
make run-verbose

# Tests
make test
```

## Adding a Task

1. Create `internal/tasks/yourtask/yourtask.go` - implement `tasks.Task` (or `tasks.HTTPTask` for HTTP routes)
2. Create `internal/tasks/yourtask/config.go` - define `Config` struct and `LoadConfig(getenv)` that returns `(*Config, error)` (`nil` when `RIG_YOURTASK_ENABLED=false`)
3. Register in `cmd/rig/main.go`:

```go
ytcfg, err := yourtask.LoadConfig(getenv)
if err != nil {
    return fmt.Errorf("yourtask config: %w", err)
}
if ytcfg != nil {
    rt.Register(yourtask.New(notifier, logger, *ytcfg))
}
```

## Deployment

Docker image is multi-stage (`golang:1.26-alpine` build, `alpine:3.23` runtime) with cross-compilation support. The CI workflow builds for `linux/amd64` and `linux/arm64`.

```bash
# Build and start
docker compose up -d --build

# Logs
docker compose logs -f

# Stop
docker compose down
```

All task state is persisted in a single JSON file (`/data/rig.json` by default) in the `rig-data` volume. Writes are atomic (temp file + rename) so a crash can never corrupt state. The GeoIP database lives in the `geoip` volume, shared read-only with the `rig` container.

## License

MIT

package tasks

import (
	"context"

	"github.com/en9inerd/go-pkgs/router"
)

type Task interface {
	Name() string
	Start(ctx context.Context) error
}

type HTTPTask interface {
	Task
	RegisterRoutes(g *router.Group)
}

// Stopper is optionally implemented by tasks that need to drain
// in-flight work after the HTTP server has stopped accepting requests.
type Stopper interface {
	Stop()
}

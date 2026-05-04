package healthcheck

import (
	"fmt"
	"net/http"
	"time"
)

func Check(addr string, useTLS bool) error {
	scheme := "http"
	if useTLS {
		scheme = "https"
	}

	url := fmt.Sprintf("%s://127.0.0.1%s/health", scheme, addr)

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("healthcheck failed: %d", resp.StatusCode)
	}

	return nil
}

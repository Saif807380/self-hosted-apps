package stores

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var defaultHTTP = &http.Client{Timeout: 20 * time.Second}

const userAgent = "trove/1.0 (+self-hosted price tracker)"

// doJSON performs req and decodes a 2xx JSON body into out.
func doJSON(ctx context.Context, c *http.Client, req *http.Request, out any) error {
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: HTTP %d: %s", req.Method, req.URL.Host, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func getJSON(ctx context.Context, c *http.Client, url string, out any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return doJSON(ctx, c, req, out)
}

func orDefault(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return defaultHTTP
}

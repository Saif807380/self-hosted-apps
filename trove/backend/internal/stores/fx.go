package stores

import (
	"context"
	"fmt"
	"net/http"
)

const fxURL = "https://api.frankfurter.dev"

// FXClient fetches ECB reference rates, used to show eShop USD prices in INR.
type FXClient struct {
	HTTP    *http.Client
	BaseURL string // overridden in tests
}

// USDToINR returns how many rupees one US dollar buys.
func (c *FXClient) USDToINR(ctx context.Context) (float64, error) {
	base := fxURL
	if c.BaseURL != "" {
		base = c.BaseURL
	}
	var resp struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := getJSON(ctx, orDefault(c.HTTP), base+"/v1/latest?base=USD&symbols=INR", &resp); err != nil {
		return 0, fmt.Errorf("fx rate: %w", err)
	}
	rate, ok := resp.Rates["INR"]
	if !ok || rate <= 0 {
		return 0, fmt.Errorf("fx rate: no INR rate in response")
	}
	return rate, nil
}

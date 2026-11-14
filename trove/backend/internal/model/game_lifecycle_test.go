package model

import "testing"

func TestValidateGameLifecycle(t *testing.T) {
	ptr := func(s string) *string { return &s }
	price := func(n int32) *int32 { return &n }
	cases := []struct {
		name     string
		status   string
		platform *string
		target   *int32
		ok       bool
	}{
		{"played, nothing else", GamePlayed, nil, nil, true},
		{"wishlist with target", GameWishlist, ptr("switch2"), price(1500), true},
		{"empty status", "", nil, nil, false},
		{"unknown status", "playing", nil, nil, false},
		{"unknown platform", GameOwned, ptr("xbox"), nil, false},
		{"zero target", GameWishlist, nil, price(0), false},
	}
	for _, c := range cases {
		err := ValidateGameLifecycle(c.status, c.platform, c.target)
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%t", c.name, err, c.ok)
		}
	}
}

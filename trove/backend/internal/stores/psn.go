package stores

import (
	"net/url"
	"regexp"
)

const psnStoreURL = "https://store.playstation.com/en-in"

// psnPath matches PS Store product and concept pages, capturing the id.
var psnPath = regexp.MustCompile(`^/[a-z]{2}-[a-z]{2}/(product|concept)/([A-Za-z0-9_-]+)/?$`)

// PSNSearchListing links to a PS Store search for the title. It's the default
// when no product URL has been pasted.
func PSNSearchListing(title string) Listing {
	return Listing{
		Store:    PSN,
		Title:    title,
		URL:      psnStoreURL + "/search/" + url.PathEscape(title),
		Region:   "IN",
		Currency: "INR",
	}
}

// PSNProductListing validates a pasted PS Store URL and returns a direct
// listing for it, rewritten to the India store.
func PSNProductListing(raw, title string) (Listing, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Host != "store.playstation.com" {
		return Listing{}, false
	}
	m := psnPath.FindStringSubmatch(u.Path)
	if m == nil {
		return Listing{}, false
	}
	return Listing{
		Store:    PSN,
		Ref:      m[2],
		Title:    title,
		URL:      psnStoreURL + "/" + m[1] + "/" + m[2],
		Region:   "IN",
		Currency: "INR",
	}, true
}

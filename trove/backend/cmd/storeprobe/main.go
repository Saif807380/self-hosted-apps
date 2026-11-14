// Command storeprobe runs the store adapters live against a few titles and
// prints what Trove would track for each. It's a debugging aid, not part of
// the server.
//
//	IGDB_CLIENT_ID=… IGDB_CLIENT_SECRET=… go run ./cmd/storeprobe "Hades II" "Elden Ring"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/stores"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal(`usage: storeprobe "title" ["title" …]`)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	igdb := &catalog.Client{ClientID: os.Getenv("IGDB_CLIENT_ID"), ClientSecret: os.Getenv("IGDB_CLIENT_SECRET")}
	steam := &stores.SteamClient{}
	eshop := &stores.EshopClient{}

	rate, err := (&stores.FXClient{}).USDToINR(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("USD→INR %.2f\n", rate)

	for _, title := range os.Args[1:] {
		if err := probe(ctx, igdb, steam, eshop, rate, title); err != nil {
			fmt.Printf("\n%s: ERROR %v\n", title, err)
		}
	}
}

func probe(ctx context.Context, igdb *catalog.Client, steam *stores.SteamClient, eshop *stores.EshopClient, rate float64, title string) error {
	found, err := igdb.Search(ctx, title, 1)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		return fmt.Errorf("no IGDB match")
	}
	g := found[0]
	fmt.Printf("\n%s (IGDB %d, %d) dev=%v\n  genres=%v themes=%v rating=%.0f similar=%d\n",
		g.Title, g.IGDBID, g.ReleaseYear, g.Developers, g.Genres, g.Themes, g.Rating, len(g.SimilarIDs))
	fmt.Printf("  platforms: PC=%t PS5=%t PS4=%t Switch=%t Switch2=%t\n",
		g.HasPlatform(catalog.PlatformPC), g.HasPlatform(catalog.PlatformPS5), g.HasPlatform(catalog.PlatformPS4),
		g.HasPlatform(catalog.PlatformSwitch), g.HasPlatform(catalog.PlatformSwitch2))

	if g.SteamAppID != "" {
		quotes, err := steam.Quotes(ctx, []string{g.SteamAppID})
		if err != nil {
			return err
		}
		printQuote("Steam "+g.SteamAppID, quotes[g.SteamAppID], 0)
	} else if g.HasPlatform(catalog.PlatformPC) {
		fmt.Println("  Steam: no appid in IGDB")
	}

	if g.HasPlatform(catalog.PlatformSwitch, catalog.PlatformSwitch2) {
		listings, err := eshop.Find(ctx, g.Title)
		if err != nil {
			return err
		}
		if len(listings) == 0 {
			fmt.Println("  eShop: no exact title match")
		}
		ids := make([]string, len(listings))
		for i, l := range listings {
			ids[i] = l.Ref
		}
		quotes, err := eshop.Quotes(ctx, ids)
		if err != nil {
			return err
		}
		for _, l := range listings {
			printQuote(fmt.Sprintf("eShop %s %s", l.Edition, l.Ref), quotes[l.Ref], rate)
		}
	}

	if g.HasPlatform(catalog.PlatformPS5, catalog.PlatformPS4) {
		fmt.Println("  PS Store:", stores.PSNSearchListing(g.Title).URL)
	}
	return nil
}

func printQuote(label string, q stores.Quote, usdToINR float64) {
	if q.Currency == "" {
		fmt.Printf("  %-28s not purchasable\n", label)
		return
	}
	line := fmt.Sprintf("  %-28s %s %.2f (base %.2f, -%d%%)", label, q.Currency,
		float64(q.FinalMinor)/100, float64(q.BaseMinor)/100, q.DiscountPct)
	if q.Currency == "USD" && usdToINR > 0 {
		line += fmt.Sprintf(" ≈ ₹%.0f", float64(q.FinalMinor)/100*usdToINR)
	}
	if q.SaleEndsAt != nil {
		line += " until " + q.SaleEndsAt.Format("2006-01-02")
	}
	fmt.Println(line)
}

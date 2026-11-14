# Trove

A personal library for the books I read, the games I play, the places I travel and my workouts. It's a PWA that works offline on my phone and syncs with a small Go server at home.

## Gaming

The Games page is also a buying guide for my three devices: the Dell G15 laptop (Linux), the PS5 and the Switch 2.

- **Lists:** Played, Wishlist, Owned and Dropped. Moving a game from the wishlist to owned to played works offline.
- **Prices:** each game's Steam (India) and Nintendo eShop (US) prices are checked every 6 hours, with a price-history chart. The PS Store gets a link and a PlatPrices link instead ([ADR 0002](docs/adr/0002-ps-store-is-link-only.md)).
- **Sale alerts:** a push notification to the phone when a wishlisted game goes on sale, reaches its target price or hits its lowest price.
- **Where to play:** a device verdict for every game, from PC requirements, ProtonDB, anti-cheat support and reviews of each version, weighed by Claude.
- **Deals:** wishlisted games on sale right now, and five weekly picks: games on sale that I'd probably like, chosen every Sunday from my ratings.

A background worker gathers all of this. It lives in server-only tables rather than the offline store ([ADR 0001](docs/adr/0001-worker-data-in-server-only-tables.md)).

## Docs

- [DEVELOPMENT.md](DEVELOPMENT.md): running it locally and in production, migrations, and the worker's settings.
- [CONTEXT.md](CONTEXT.md): the words Trove uses (listing, price run, deal, pick, …) and what they mean.
- [docs/adr](docs/adr): decisions worth knowing before changing things.

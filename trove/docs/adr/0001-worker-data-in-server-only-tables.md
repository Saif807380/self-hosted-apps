# Worker-generated gaming data lives in server-only tables, not in offline sync

Trove's PWA keeps your records in IndexedDB and syncs them with the server, so you can use it offline. The gaming worker produces a lot more than your records: IGDB matches, store listings, price history, sale alerts, device verdicts and weekly picks. All of that lives in server-only tables (`game_meta`, `store_listings`, `price_quotes`, `sale_events`, `device_verdicts`, `picks`) that are never synced. The UI fetches it through `GamingService` when online and says "needs a connection" when not.

The data is large and read-only for you, it goes stale without a connection anyway (a price from last week isn't one you can buy at), and keeping it out of sync means the worker can't fight an edit you made offline.

## Consequences

- Only your own choices sync: a game's status, target price and device live on `video_games`, so you can move a game from wishlist to owned on a train.
- The worker writes to `video_games` only in narrow, one-way cases: it fills in a missing cover or studio, takes IGDB's title when it first links a game, and clears those filled-in values when you change the match. It never overwrites something you set. Each write moves `updated_at` so your devices pull it.
- The Deals tab, prices, verdicts and picks don't work offline. That's accepted: they're only useful when you can act on them.

## Considered options

- **Sync everything to IndexedDB**: rejected. Price history and verdicts would bloat every device, and sync conflicts between the worker and offline edits would need resolving.
- **Store the worker's results as columns on `video_games`**: rejected for the same conflict reason, and because one game has many listings and price runs.

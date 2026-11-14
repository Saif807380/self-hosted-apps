# The PS Store is a link, not a priced store

Trove tracks prices on Steam and the Nintendo eShop, but for the PS Store it only links to the game's store page (or a store search) and to PlatPrices, which shows PlayStation prices and price history. Sony has no public price API, and pricing it would mean calling the PS Store's private GraphQL API with copied query hashes, which is fragile and was ruled out. You already use the PlatPrices browser extension on the PS Store page, so a link gets you the same information without Trove depending on Sony's internals.

## Consequences

- PS Store games raise no sale alerts and never show up as deals or picks.
- If PlatPrices offers an API key one day, PS prices can be added to the same `psn` listings that hold the links today.

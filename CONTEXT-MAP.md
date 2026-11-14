# Context Map

Each app in this repo is its own context with its own language.

## Contexts

- [Trove](./trove/CONTEXT.md): personal library (books, games, travel, workouts) and a buying guide for games

## Relationships

- **Trove → cortex**: Trove's worker searches cortex's SearXNG (`127.0.0.1:8888`) for game reviews when making device verdicts, which is why SearXNG starts on boot. Trove works without it; verdicts made while it's down are redone later.

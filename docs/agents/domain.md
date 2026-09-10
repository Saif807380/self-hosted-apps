# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

This repo is **multi-context**: it holds several independent apps (`pitwall`, `streamcloud`, `trove`) rather than one shared codebase, so each gets its own domain doc instead of a single root one.

## Before exploring, read these

- **`CONTEXT-MAP.md`** at the repo root: points at one `CONTEXT.md` per context. Read each one relevant to the topic.
- **`docs/adr/`** at the repo root: system-wide decisions that span more than one app.
- **`<app>/CONTEXT.md`** (e.g. `pitwall/CONTEXT.md`, `streamcloud/CONTEXT.md`, `trove/CONTEXT.md`): the glossary for that app.
- **`<app>/docs/adr/`**: decisions scoped to that app.

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest creating them upfront. The `/domain-modeling` skill (reached via `/grill-with-docs` and `/improve-codebase-architecture`) creates them lazily when terms or decisions actually get resolved.

## File structure

```
/
├── CONTEXT-MAP.md
├── docs/adr/                  ← system-wide decisions
├── pitwall/
│   ├── CONTEXT.md
│   └── docs/adr/              ← pitwall-specific decisions
├── streamcloud/
│   ├── CONTEXT.md
│   └── docs/adr/              ← streamcloud-specific decisions
└── trove/
    ├── CONTEXT.md
    └── docs/adr/              ← trove-specific decisions
```

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in the relevant app's `CONTEXT.md`. Don't drift to synonyms the glossary explicitly avoids.

If the concept you need isn't in the glossary yet, that's a signal: either you're inventing language the project doesn't use (reconsider) or there's a real gap (note it for `/domain-modeling`).

## Flag ADR conflicts

If your output contradicts an existing ADR (root or app-scoped), surface it explicitly rather than silently overriding:

> _Contradicts ADR-0007 (event-sourced orders), but worth reopening because…_

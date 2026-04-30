# streamcloud

Self-hosted media stack on a single Arch laptop. Replaces Netflix / Prime / Hotstar / Crunchyroll / YT Music / Google Photos.

Phase 1 (video) and Phase 3 (music) are live. Phases 2 and 4 (sports / photos) are planned in `~/.claude/plans/implement-a-plan-for-delegated-salamander.md`.

---

## Ports — bookmark these

All bound to `127.0.0.1` on the laptop. From the phone or another device, replace `localhost` with the laptop's Tailscale IP (`tailscale ip -4`).

| Service        | URL                       | What it does                                         |
|----------------|---------------------------|------------------------------------------------------|
| Jellyfin       | http://localhost:8096     | Watch movies/TV (web UI, also via Jellyfin Media Player / Findroid) |
| Navidrome      | http://localhost:4533     | Music server (Subsonic API; also via Feishin / Tempo on Android) |
| qBittorrent    | http://localhost:8080     | Torrent client WebUI (runs inside the VPN namespace) |
| Sonarr         | http://localhost:8989     | TV show automation                                   |
| Radarr         | http://localhost:7878     | Movie automation                                     |
| Prowlarr       | http://localhost:9696     | Indexer manager (feeds Sonarr/Radarr and FLAC searches) |
| Bazarr         | http://localhost:6767     | Subtitle automation                                  |
| FlareSolverr   | http://localhost:8191     | Cloudflare-bypass proxy (used by Prowlarr only)      |

Default admin user for the *arr apps was set during initial setup; credentials live in your password manager (not in this repo).

---

## After a reboot — what to do

**Short answer: nothing, once you've done the one-time setup below.** Open the laptop, log in, wait ~30 seconds, and everything is back.

### What auto-starts (verified)

| Component                          | How it starts                          |
|------------------------------------|----------------------------------------|
| systemd-resolved (DNS-over-TLS)    | system service, enabled                |
| Tailscale                          | `tailscaled.service`, enabled          |
| Sonarr / Radarr / Prowlarr / Bazarr| systemd services, enabled              |
| Jellyfin                           | `jellyfin.service`, enabled            |
| Navidrome                          | `navidrome.service`, enabled           |
| gluetun + qBittorrent + FlareSolverr | rootless Podman, see one-time setup ↓ |

### One-time setup (do this once, then forget)

Rootless Podman containers don't auto-start by default. Enable the user-level "restart" service so `restart: unless-stopped` in `compose/torrent-stack.yml` actually fires on boot:

```bash
loginctl enable-linger $USER                      # already done — verifies with: loginctl show-user $USER | grep Linger
systemctl --user enable podman-restart.service    # makes containers come back after reboot
```

After this, `compose_gluetun_1`, `compose_qbittorrent_1`, and `compose_flaresolverr_1` come up automatically when you log in.

### Verify after a reboot

```bash
# All containers up?
podman ps

# VPN exit IP (should be ProtonVPN, NOT your real IP)
podman exec compose_qbittorrent_1 curl -s https://am.i.mullvad.net/json | grep -E 'ip|country'

# Tailscale up?
tailscale status | head -3

# DNS bypass active? (Servers should be 1.1.1.1, not 192.168.1.1)
resolvectl status | grep -A1 "DNS Servers"

# *arr apps reachable?
for p in 8096 8989 7878 9696 6767; do curl -fsS -o /dev/null -w "$p: %{http_code}\n" http://localhost:$p; done
```

### If the torrent stack didn't come up

```bash
cd ~/Projects/self-hosted-apps/streamcloud/compose
podman-compose --env-file ../.env -f torrent-stack.yml up -d
```

### If Radarr/Sonarr can't reach indexers (DNS poisoning)

Airtel intercepts DNS for torrent hostnames and returns `13.127.247.216`. The `systemd-resolved` config at `/etc/systemd/resolved.conf.d/dns.conf` forces DNS-over-TLS to Cloudflare to bypass it. If indexer search fails after a reboot:

```bash
# Confirm DoT is active
resolvectl status | grep DNSOverTLS                    # should show +DNSOverTLS
# Confirm a known-poisoned host resolves to its real IP
resolvectl query 1337x.to                              # should NOT be 13.127.247.216

# If broken, restart resolved
sudo systemctl restart systemd-resolved
```

---

## Where the VPN is actually used

**Only qBittorrent traffic goes through ProtonVPN.** Nothing else.

```
                   ┌─────────────────────────────────────────┐
Sonarr/Radarr  ──► │ Host network (your ISP, DoT-protected) │ ──► indexer HTTP APIs
Prowlarr       ──► │                                         │ ──► (1337x, EZTV, Nyaa, etc.)
FlareSolverr   ──► └─────────────────────────────────────────┘
                                                                
                   ┌─────────────────────────────────────────┐
qBittorrent    ──► │ gluetun container (ProtonVPN WireGuard) │ ──► tracker + peer traffic
                   │  + kill-switch + port forwarding        │     (this is the only thing
                   └─────────────────────────────────────────┘      that needs hiding)
```

This is intentional — and standard practice for the *arr stack:

- **Indexer search** is just HTTPS API calls to public-ish search sites. They don't care who's asking. Routing them through the VPN slows them down for no security gain.
- **The actual peer-to-peer torrent traffic** is what your ISP can see and what trackers log. That's the only thing that has to be VPN-hidden, and it is — qBittorrent is `network_mode: "service:gluetun"`, which means it has *no* network of its own and can only talk through the VPN container. If gluetun stops, qBittorrent loses all network. That's the kill-switch.
- **DNS-over-TLS** to Cloudflare runs at the host level, separate from the VPN. It exists because Airtel poisons DNS for torrent indexer hostnames, not because of any privacy concern.

To re-verify the kill-switch is intact:

```bash
podman stop compose_gluetun_1
podman exec compose_qbittorrent_1 curl --max-time 5 https://1.1.1.1   # MUST FAIL
podman start compose_gluetun_1
```

---

## When Radarr search returns nothing (and you can find the torrent yourself)

This is almost always a quality-profile filter, not a "no torrents exist" problem. Radarr finds the releases — then drops every one that fails its rules — then tells you "no results."

### The usual culprits, in order of likelihood

1. **Max-size cap.** `HD-1080p` profile is set to **max 5 GB / movie**. A typical 1080p remux is 8–25 GB; a 1080p HEVC encode is ~3–5 GB but only if the release group did a tight encode. Action: in **Radarr → Profiles → HD-1080p**, raise the size cap, *or* switch the movie to the `4K-favourites` profile (15 GB cap).
2. **Minimum quality cutoff.** Profiles reject anything below 1080p. If the only available releases for an old/obscure movie are 720p, Radarr ignores them all. Action: temporarily allow 720p in the profile, re-search, then revert.
3. **Custom format score.** HEVC/x265 gets +10 in custom formats. If only x264 releases exist, score may dip below the "minimum format score" threshold. Action: lower the minimum score in the profile, or accept x264.
4. **Indexer category coverage.** EZTV is TV-only; Nyaa is anime-only. If only those two are tagged for movies, Radarr finds nothing. Action: in **Prowlarr → Indexers**, confirm at least one general-purpose indexer (1337x, TheRARBG) is tagged with the movie category.
5. **Indexer rate-limited (429).** After consecutive failures Prowlarr disables the indexer for ~1 minute. Action: wait 60s and re-search; check **Prowlarr → System → Events** for "indexer disabled" messages.
6. **Movie not actually released yet.** Trakt watchlists pick up "announced" titles. If physical/streaming release hasn't happened, no torrent exists anywhere. Action: check release date on TMDB.

### Diagnose what Radarr saw vs. what it rejected

```
Radarr → (movie) → Manual Search   # shows ALL releases the indexers returned
                                   # rejected ones have a red icon + reason
```

If the manual search list shows the release you found by hand, the rejection reason next to it tells you exactly which rule killed it. Click it to grab manually anyway (overrides profile).

---

## What runs at startup vs. what you launch yourself

### Always running (no action needed)

- All servarr apps, Jellyfin, qBittorrent, gluetun, FlareSolverr, Tailscale, DNS-over-TLS resolver.

### Launch on demand

- **Jellyfin Media Player** — laptop video playback (`jellyfin-media-player`).
- **Findroid / Streamyfin** — phone Jellyfin clients.
- **Browser tabs** — for Sonarr/Radarr/Prowlarr/Bazarr/qBittorrent admin (the bookmarks above).

That's it — there's no daily start-up routine. The stack is "always on" while the laptop is on.

---

## Filesystem layout

```
/srv/media/
├── video/
│   ├── movies/        # Radarr root folder
│   ├── tv/            # Sonarr root folder
│   ├── anime/         # Sonarr root folder (separate)
│   └── sports/        # Phase 2 (yt-dlp / streamlink)
├── music/             # Phase 3 (Navidrome)
├── photos/            # Phase 4 (Immich)
├── photos-orig/       # Phase 4 staging
└── downloads/         # qBittorrent landing dir (CoW disabled, hardlinked into above)
```

Hardlink imports from `/srv/media/downloads` into `/srv/media/video/...` mean a file lives in both places without using double the disk.

---

## Project layout

```
streamcloud/
├── compose/
│   └── torrent-stack.yml         # gluetun + qBittorrent + FlareSolverr
├── config/
│   └── qbit/                     # qBittorrent config (gitignored — has creds/state)
├── docs/                         # Blog post drafts (see plan)
├── quadlets/                     # systemd quadlets (planned migration target)
├── scripts/                      # prune-video.sh, ytm-csv-to-list.sh, music-rip-opus.sh
├── systemd-user/                 # User-level timers (sports-grab, disk-prune)
├── .env                          # Secrets — chmod 600, gitignored
├── .env.example                  # Template
└── .gitignore
```

---

## Music (replaces YouTube Music)

**Stack:** Navidrome (server) → Feishin (laptop) → PixelPlay (Android, beta).

Default format is **Opus 192k** — transparent for casual listening, ~50 MB/album, fits ~600 albums in the 30 GB pool. FLAC is opt-in for select favourites where bit-perfect matters.

### Adding new music going forward

**One-shot wrapper (preferred):**
```bash
./scripts/add-music.sh '<youtube-url>' ['<another-url>' ...]
```
Downloads to `~/Music/` as Opus 192k, then prompts to run `beet import` which moves files into `/srv/media/music/<albumartist>/<album>/`. Accepts single tracks, album playlists, or YT Music playlists.

**Batch (large list):** append URLs to `config/ytm-library.txt`, then `./scripts/music-rip-opus.sh`. The archive file deduplicates, so re-runs are safe.

**FLAC for select favourites:** search `Artist Album FLAC` in Prowlarr (`:9696` → Search), send to qBittorrent with category `music`, then move into `/srv/media/music/<artist>/<album>/`. Only worth the manual workflow for genuine "I want bit-perfect" picks.

### Tagging with beets

After any batch import:
```bash
beet import /srv/media/music
```
beets matches against MusicBrainz, normalises filenames, embeds cover art. Config at `~/.config/beets/config.yaml`.

### Storage budget

30 GB cap on `/srv/media/music`. Rough capacity:
- Opus 192k: ~50 MB/album → ~600 albums
- FLAC: ~300 MB/album → ~100 albums

A typical mix is 95% Opus with a handful of FLAC favourites.

### Clients

| Where  | App      | Notes                                                |
|--------|----------|------------------------------------------------------|
| Laptop | Feishin  | `feishin` — connect to `http://localhost:4533`       |
| Pixel  | Tempo    | Play Store — connect to `http://<tailscale-ip>:4533` |

Both clients support downloading albums for offline playback (hit the download icon while the laptop is on; then it works on LTE with the laptop off).

### Smart playlists

Five Navidrome Smart Playlist (`.nsp`) files live in `/srv/media/music/Playlists/` and appear automatically in Feishin/Tempo:

| Playlist        | Rule                                          |
|-----------------|-----------------------------------------------|
| Recently Added  | Tracks added in the last 30 days, sorted desc |
| Top Played      | Top 100 by play count                         |
| Unplayed        | 50 random tracks with playCount = 0           |
| Loved           | Anything you've starred / hearted             |
| Random Mix      | 50 random tracks (refreshes on open)          |

Edit the `.nsp` JSON files directly to tweak rules. Format reference: [Navidrome smart playlists](https://github.com/navidrome/navidrome/blob/master/tests/fixtures/playlists/recently_played.nsp). Field names and operators come from `model/criteria/{fields,operators}.go` in the Navidrome repo.

### Discovery via Last.fm + ListenBrainz

Once you wire credentials into Navidrome (per-user Settings → Personal), every play is scrobbled to both services. After ~2–3 weeks of scrobbles, `troi` generates personalised playlists (Daily Jams, Weekly Exploration). The `scripts/generate-listenbrainz-playlists.sh` runner converts troi's JSPF output to M3U via Navidrome's Subsonic search, dropping `lb-*.m3u` into the Playlists folder. Scheduled daily at 06:00 by `systemd-user/generate-playlists.timer`.

### Verify Navidrome

```bash
systemctl is-active navidrome                        # active
curl -sI http://localhost:4533/ | head -1            # HTTP/1.1 ...
ls /srv/media/music/Playlists/                       # 5 .nsp files + lb-*.m3u once troi runs
```

For the one-time Google Takeout import flow (already done), see Phase 3 in `~/.claude/plans/implement-a-plan-for-delegated-salamander.md`.

---

## References

- Plan: `~/.claude/plans/implement-a-plan-for-delegated-salamander.md`
- gluetun docs: https://github.com/qdm12/gluetun-wiki
- Servarr docs: https://wiki.servarr.com/

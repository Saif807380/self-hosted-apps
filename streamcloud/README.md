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

### If torrents stall at "downloading metadata"

Symptom: Sonarr/Radarr hand a magnet to qBittorrent and it sits at `metaDL` forever, 0 peers, even on a huge swarm.

**Root cause seen in practice:** qBittorrent was bound to *all* interfaces (`Session\Interface` empty), so libtorrent listened on both `tun0` (the VPN) **and** `eth0` (the podman bridge). gluetun's firewall drops everything sourced from `eth0` — outbound UDP there fails instantly with `EPERM`. That kills DHT bootstrap: `dht_nodes` sits at **0** permanently, and with no DHT the client depends entirely on trackers, which is not enough to find peers for a magnet.

qBittorrent **must be pinned to the VPN interface**. This lives in `config/qbit/` (gitignored), so it is not restored by cloning the repo:

```
qBittorrent → Settings → Advanced → Network Interface → tun0
```

Or via the WebUI API:

```bash
curl -s -d 'json={"current_network_interface":"tun0"}' \
  http://localhost:8080/api/v2/app/setPreferences
podman restart compose_qbittorrent_1     # libtorrent needs a restart to rebuild DHT
```

Verify — `dht_nodes` should climb into the hundreds within a minute, and sockets should show **only** `10.2.0.2` (tun0), never `10.89.0.2` (eth0):

```bash
curl -s http://localhost:8080/api/v2/transfer/info      # dht_nodes > 0
podman exec compose_gluetun_1 netstat -tuln | grep -v 8080
```

**Related:** ProtonVPN rotates the forwarded port on every reconnect, but qBittorrent stores a static `Session\Port`. When they drift, inbound peers are firewalled off. `scripts/sync-qbit-port.sh` is wired to gluetun's `VPN_PORT_FORWARDING_UP_COMMAND` to push each new port into qBittorrent automatically.

### If every indexer is "disabled for 24 hours"

Symptom: Sonarr/Radarr show all indexers disabled, health warns *"Indexers unavailable due to failures for more than 6 hours"*, and it never recovers on its own.

**There is usually no single broken indexer.** Two independent backoff ladders amplify ordinary flakiness into a permanent-looking outage:

1. Prowlarr keeps a per-indexer failure ladder that doubles up to a 24h cap. While an indexer is backed off, its Torznab endpoint returns `429` in ~2ms **without trying upstream**.
2. Sonarr/Radarr read that `429` as "API Request Limit reached" and apply *their own* doubling ladder on top.

Either ladder only resets on a **successful** request, so once both are at the cap nothing retries often enough to recover. Check how long they've actually been failing — `initialFailure` is the honest signal:

```bash
curl -s -H "X-Api-Key: $KEY" http://localhost:9696/api/v1/indexerstatus
```

**Root cause seen in practice: IPv6 broke FlareSolverr.** Prowlarr does not use FlareSolverr's solved HTML — it takes the `cf_clearance` cookie and *replays* the request from its own HTTP stack. That cookie is bound to the IP it was issued to. FlareSolverr runs in a rootless podman container with no IPv6, so it egresses via the host's IPv4, while .NET's Happy Eyeballs made Prowlarr replay over IPv6 from a completely different address. Cloudflare answered `403` every time, and every Cloudflare-fronted indexer failed no matter how well FlareSolverr worked.

Replaying one cookie from each address family isolates it:

```
IPv4: http=200      # same egress as FlareSolverr
IPv6: http=403      # different address, cookie rejected
```

Fix — force Prowlarr's outbound onto IPv4 (this does **not** affect the WebUI listener, which stays dual-stack):

```bash
sudo install -Dm644 systemd/prowlarr.service.d/10-disable-ipv6.conf \
  /etc/systemd/system/prowlarr.service.d/10-disable-ipv6.conf
sudo systemctl daemon-reload && sudo systemctl restart prowlarr
```

Then clear the ladders — a successful **Test** is the only thing that resets them, in Prowlarr *and* again in Sonarr/Radarr. Expect to retry: the ISP resets connections intermittently on both address families (~1/10 for `1337x.to`, ~5/10 for `eztvx.to`), so a single failed test means nothing.

**What this does not fix:** indexers failing for their own reasons. `apibay.org` (The Pirate Bay) serves a Cloudflare *"Rate Limited"* page that reproduces from plain curl, and `nyaa.si` is IPv4-only with no Cloudflare, so FlareSolverr never engages and ISP interference hits it directly. Those recover on their own once the ladders are no longer pinned at 24h.

### If subtitles never appear

Bazarr handles this automatically — it watches Sonarr/Radarr and fetches subtitles for anything that arrives without them. When nothing shows up, check that it can *write* before blaming the providers:

```bash
id bazarr                                    # must include the media group
grep -c PermissionError /var/lib/bazarr/log/bazarr.log
```

**Root cause seen in practice:** the Arch `bazarr` package ships `Group=bazarr`, while `sonarr`/`radarr`/`prowlarr` all ship `Group=media`. `/srv/media` is `drwxrwsr-x … media`, so Bazarr could read the library but every single save failed:

```
BAZARR Error saving Subtitles file to disk … PermissionError(13, 'Permission denied'):
'… The Flash (2014) - S01E14 - Fallout Bluray-1080p.en.srt'
```

This is worse than it looks. Bazarr **downloads a subtitle before it writes it**, so each failed save still spent one OpenSubtitles download. It burned the ~20/day free-tier cap on files it then discarded, tripped `DownloadLimitExceeded`, got throttled 6 hours, and repeated — daily, silently, for weeks. Fix:

```bash
sudo install -Dm644 systemd/bazarr.service.d/10-media-group.conf \
  /etc/systemd/system/bazarr.service.d/10-media-group.conf
sudo systemctl daemon-reload && sudo systemctl restart bazarr
```

The media directories are setgid, so new subtitles inherit group `media` on their own.

**Backfilling won't work until you clear adaptive search.** After 3 weeks of failed attempts Bazarr drops an item to one retry per week, so the scheduled search skips it — silently, because that log line is DEBUG and `debug` is off. A long-broken library is therefore *fully* adaptive-throttled and a normal search finishes in seconds having done nothing. Turn `adaptive_searching` off for one pass, run both wanted-search tasks, then turn it back on.

Also reset the provider throttles first (**Settings → Providers → Reset**), or providers stay disabled from failures that are already fixed.

**On providers:** the free OpenSubtitles tier is ~20 downloads/day and `yifysubtitles` is movies-only, so TV had exactly one usable source. Also enabled: `gestdown` (Addic7ed mirror, strong on TV), `tvsubtitles`, `subf2m`, and `embeddedsubtitles` (extracts tracks already inside the mkv — no network, no quota).

Two gotchas found the hard way:

- **`subf2m` needs a user-agent set** or it throttles itself for 12 hours with `ConfigurationError('User-agent config missing')`. Its default is an empty string, so enabling the provider is not enough.
- **`use_embedded_subs: true`** makes Bazarr treat an existing embedded track as "not missing", so the `embeddedsubtitles` provider only fires on files it has already decided it wants.

`subsource` and `subdl` both require a free API key. Bazarr's manual per-episode download API (`PATCH /api/episodes/subtitles`) returns 204 but silently fails when the episode's profile id is falsy — `get_profiles_list` hands back the whole list and `download.py` subscripts it by name. Use the wanted-search tasks instead.

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

- **Indexer search** is just HTTPS API calls to public-ish search sites. They don't care who's asking. Routing them through the VPN slows them down for no security gain — and measurably *hurts*: from the VPN exit, `1337x.to` and `eztvx.to` return an immediate `403` instead of a solvable challenge, and `apibay.org` times out. The one thing it does help is `nyaa.si` (3/10 direct vs 7/8 over VPN), which is not worth moving the stack for.
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

Default format is **Opus 192k** — transparent for casual listening, ~50 MB/album, fits ~600 albums in the 30 GB pool.

### Adding new music (Autonomous Pipeline)

New music is added via a 3-stage autonomous pipeline:
1. **Download:** `yt-dlp` fetches audio as Opus 192k.
2. **LLM Correction:** `fix_tags.py` uses Gemini (3-Flash) to research correct metadata, strip junk text (e.g., "[Official Video]"), and normalize "Artist - Title" strings.
3. **Tag & Move:** Metadata is embedded via Mutagen, and files are moved to `/srv/media/music/Artist/Album/Track.opus`.

**Usage:**
```bash
./scripts/add-music.sh '<youtube-url>' ['<another-url>' ...]
```
The process is fully autonomous (`--approval-mode auto_edit`). Just run the command and the music appears in Navidrome.

### Maintenance & Deletion

| Script | Purpose |
|--------|---------|
| `scripts/empty-trash.py` | **UI-based deletion.** Add tracks to the "Trash" playlist in Navidrome/Feishin. This script deletes the files from disk and purges them from the database. |
| `scripts/bulk_clean/fix_library_inplace.py` | **Fast Sync.** Syncs metadata for existing library files based *only* on their folder structure (`Artist/Album`). High-speed, no LLM required. |
| `scripts/bulk_clean/clean_lib.py` | **Batch Cleanup.** Uses LLM to clean up metadata for large batches of existing files. |

**Automatic Cleanup:**
A systemd user timer (`empty-trash.timer`) runs `empty-trash.py` daily at 10:00 AM. To delete music, simply **add it to the "Trash" playlist** and it will be gone by the next morning.

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

Both clients support downloading albums for offline playback.

### Smart playlists

Navidrome Smart Playlists (`.nsp`) live in `/srv/media/music/Playlists/`:

| Playlist        | Rule                                          |
|-----------------|-----------------------------------------------|
| Trash           | **DELETION QUEUE.** Files added here are deleted daily. |
| Recently Added  | Tracks added in the last 30 days, sorted desc |
| Top Played      | Top 100 by play count                         |
| Unplayed        | 50 random tracks with playCount = 0           |
| Loved           | Anything you've starred / hearted             |
| Random Mix      | 50 random tracks (refreshes on open)          |

Edit the `.nsp` JSON files directly to tweak rules. Format reference: [Navidrome smart playlists](https://github.com/navidrome/navidrome/blob/master/tests/fixtures/playlists/recently_played.nsp). Field names and operators come from `model/criteria/{fields,operators}.go` in the Navidrome repo.

### Discovery & Playlists

**1. Last.fm Daily Discovery (New Music)**
A custom Python script (`scripts/lastfm-discovery.py`) runs daily at 05:00 AM via a systemd timer. It:
- Fetches your recent Last.fm listening history.
- Queries Last.fm for similar tracks.
- Filters out tracks you already have in Navidrome.
- Autonomously downloads 20 brand new tracks.
- Adds them to `lastfm-discovery.m3u` in your Playlists folder.

**2. ListenBrainz Auto-Playlists (Existing Library)**
Once you wire credentials into Navidrome (per-user Settings → Personal), every play is scrobbled to both services. After ~2–3 weeks of scrobbles, `troi` generates personalised playlists (Daily Jams, Weekly Exploration) focusing on rediscovering music already in your library. The `scripts/generate-listenbrainz-playlists.sh` runner drops these into the Playlists folder. Scheduled daily at 06:00 by `systemd-user/generate-daily-playlists.timer`.

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

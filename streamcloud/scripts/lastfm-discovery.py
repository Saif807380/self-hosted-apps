#!/usr/bin/env python3
import hashlib
import json
import os
import random
import secrets
import subprocess
import sys
import time
import urllib.parse
import urllib.request
from pathlib import Path

# --- CONFIGURATION ---
SCRIPT_DIR = Path(__file__).parent
PROJECT_ROOT = SCRIPT_DIR.parent
MUSIC_FOLDER = os.environ.get("MUSIC_FOLDER", "/srv/media/music").rstrip("/")
PLAYLIST_DIR = Path(MUSIC_FOLDER) / "Playlists"
TARGET_NEW_TRACKS = 20


# Load environment variables
def load_env():
    env_file = PROJECT_ROOT / ".env"
    if env_file.exists():
        with open(env_file) as f:
            for line in f:
                if "=" in line and not line.startswith("#"):
                    k, v = line.strip().split("=", 1)
                    os.environ[k] = v.strip("\"'")


load_env()

LAST_FM_API_KEY = os.environ.get("LAST_FM_API_KEY")
LAST_FM_USER = os.environ.get(
    "LISTENBRAINZ_USER"
)  # Using same username as ListenBrainz for Last.fm
NAVIDROME_URL = os.environ.get("NAVIDROME_URL", "http://127.0.0.1:4533").rstrip("/")
NAVIDROME_USER = os.environ.get("NAVIDROME_USER")
NAVIDROME_PASS = os.environ.get("NAVIDROME_PASS", "")

if not all([LAST_FM_API_KEY, LAST_FM_USER, NAVIDROME_USER, NAVIDROME_PASS]):
    print("Error: Missing required API keys or credentials in .env")
    sys.exit(1)


# --- LAST.FM HELPERS ---
def lastfm_call(method: str, **params) -> dict:
    base = {"method": method, "api_key": LAST_FM_API_KEY, "format": "json"}
    qs = urllib.parse.urlencode({**base, **params})
    url = f"http://ws.audioscrobbler.com/2.0/?{qs}"

    try:
        req = urllib.request.Request(
            url, headers={"User-Agent": "streamcloud-discovery/1.0"}
        )
        with urllib.request.urlopen(req, timeout=10) as r:
            return json.loads(r.read())
    except Exception as e:
        print(f"Last.fm Error ({method}): {e}")
        return {}


def get_recent_tracks(limit=50):
    data = lastfm_call("user.getrecenttracks", user=LAST_FM_USER, limit=limit)
    tracks = data.get("recenttracks", {}).get("track", [])

    # Filter out "now playing" which has a different structure
    return [t for t in tracks if not t.get("@attr", {}).get("nowplaying")]


def get_similar_tracks(artist: str, track: str, limit=10):
    data = lastfm_call(
        "track.getsimilar", artist=artist, track=track, limit=limit, autocorrect=1
    )
    return data.get("similartracks", {}).get("track", [])


# --- NAVIDROME HELPERS ---
def subsonic_call(endpoint: str, **params) -> dict:
    salt = secrets.token_hex(8)
    token = hashlib.md5((NAVIDROME_PASS + salt).encode()).hexdigest()
    base = {
        "u": NAVIDROME_USER,
        "t": token,
        "s": salt,
        "v": "1.16.1",
        "c": "streamcloud-discovery",
        "f": "json",
    }
    qs = urllib.parse.urlencode({**base, **params})
    url = f"{NAVIDROME_URL}/rest/{endpoint}.view?{qs}"
    try:
        with urllib.request.urlopen(url, timeout=10) as r:
            resp = json.loads(r.read())["subsonic-response"]
            if resp.get("status") == "failed":
                print(f"Subsonic Error: {resp.get('error', {}).get('message')}")
                return {}
            return resp
    except Exception as e:
        print(f"Request Error: {e}")
        return {}


def check_in_library(artist: str, title: str) -> bool:
    """Returns True if the track is found in the local library."""
    query = f"{artist} {title}"
    resp = subsonic_call(
        "search3", query=query, songCount=1, artistCount=0, albumCount=0
    )
    songs = resp.get("searchResult3", {}).get("song", [])
    return len(songs) > 0


def get_file_path(artist: str, title: str) -> str | None:
    """Gets the absolute file path for a track from Navidrome."""
    query = f"{artist} {title}"
    resp = subsonic_call(
        "search3", query=query, songCount=1, artistCount=0, albumCount=0
    )
    songs = resp.get("searchResult3", {}).get("song", [])
    if not songs:
        return None
    path = songs[0].get("path")
    if not path:
        return None
    return f"{MUSIC_FOLDER}/{path}"


# --- MAIN WORKFLOW ---
def main():
    print(f"--- Starting Last.fm Discovery for user '{LAST_FM_USER}' ---")

    print("1. Fetching recent tracks...")
    recent = get_recent_tracks(limit=50)
    if not recent:
        print("No recent tracks found. Exiting.")
        return

    print(f"   Found {len(recent)} recent plays.")

    # Shuffle to ensure variety in recommendations each day
    random.shuffle(recent)

    candidates = []
    print("2. Gathering similar tracks (building candidate pool)...")
    for track in recent[:15]:  # Use up to 15 recent tracks as seeds
        artist = track.get("artist", {}).get("#text")
        title = track.get("name")
        if not artist or not title:
            continue

        similar = get_similar_tracks(artist, title, limit=10)
        for s_track in similar:
            s_artist = s_track.get("artist", {}).get("name")
            s_title = s_track.get("name")
            if s_artist and s_title:
                candidates.append({"artist": s_artist, "title": s_title})

        # To avoid spamming the API, brief pause
        time.sleep(0.5)

    # Deduplicate candidates
    unique_candidates = []
    seen = set()
    for c in candidates:
        key = f"{c['artist']}-{c['title']}".lower()
        if key not in seen:
            seen.add(key)
            unique_candidates.append(c)

    # Shuffle candidates to get a random mix
    random.shuffle(unique_candidates)

    print(f"   Found {len(unique_candidates)} unique candidate tracks.")

    print("3. Filtering against local Navidrome library...")
    to_download = []
    for c in unique_candidates:
        if len(to_download) >= TARGET_NEW_TRACKS:
            break

        if not check_in_library(c["artist"], c["title"]):
            print(f"   [NEW] {c['artist']} - {c['title']}")
            to_download.append(c)
        else:
            print(f"   [SKIP] Already in library: {c['artist']} - {c['title']}")

    if not to_download:
        print("No new tracks found to download. Exiting.")
        return

    print(f"\n4. Downloading {len(to_download)} new tracks via addition pipeline...")
    add_music_script = SCRIPT_DIR / "add-music.sh"

    search_queries = []
    for i, t in enumerate(to_download):
        print(f"   [{i + 1}/{len(to_download)}] Queuing: {t['artist']} - {t['title']}")
        search_queries.append(f"ytsearch1:{t['artist']} - {t['title']}")

    env = os.environ.copy()
    env["AUTO_IMPORT"] = "1"

    try:
        # We call the script with all queries at once to batch the yt-dlp downloads
        # and ensure fix_tags.py (LLM pipeline) only runs exactly once.
        subprocess.run([str(add_music_script)] + search_queries, env=env, check=False)
    except Exception as e:
        print(f"   Error running addition pipeline: {e}")

    print("\n5. Generating Discovery Playlist...")
    print("   Triggering Navidrome rescan...")
    subsonic_call("startScan")
    print("   Waiting 30 seconds for database update...")
    time.sleep(30)

    PLAYLIST_DIR.mkdir(parents=True, exist_ok=True)
    m3u_path = PLAYLIST_DIR / "lastfm-discovery.m3u"

    found_count = 0
    with open(m3u_path, "w") as f:
        f.write("#EXTM3U\n")
        f.write("#PLAYLIST:Newly Discovered\n")
        f.write("#EXTART:Generated from recent listening history\n")

        for t in to_download:
            path = get_file_path(t["artist"], t["title"])
            # Fallback fuzzy match if exact search fails after LLM correction
            if not path:
                # Try just the title
                path = get_file_path("", t["title"])

            if path:
                f.write(f"#EXTINF:-1,{t['artist']} - {t['title']}\n")
                f.write(f"{path}\n")
                found_count += 1
            else:
                print(
                    f"   Could not resolve final path for {t['artist']} - {t['title']}"
                )

    print(f"\n🚀 Discovery complete! Added {found_count} tracks to {m3u_path.name}")


if __name__ == "__main__":
    main()

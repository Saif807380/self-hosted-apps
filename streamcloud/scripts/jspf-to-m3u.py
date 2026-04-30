#!/usr/bin/env python3
"""
Convert a troi JSPF playlist to an M3U pointing at files in your Navidrome library.

For each JSPF track, search Navidrome's Subsonic API by artist + title, take the
first match, and emit an absolute file path. Tracks not found in the library are
skipped (with a stderr note).

Reads JSPF from stdin or a file argument. Writes M3U to stdout.

Required env vars (read from streamcloud/.env via the shell wrapper):
    NAVIDROME_URL   default http://127.0.0.1:4533
    NAVIDROME_USER
    NAVIDROME_PASS
    MUSIC_FOLDER    default /srv/media/music
"""
import hashlib
import json
import os
import secrets
import sys
import urllib.parse
import urllib.request


NAVIDROME_URL = os.environ.get("NAVIDROME_URL", "http://127.0.0.1:4533").rstrip("/")
NAVIDROME_USER = os.environ["NAVIDROME_USER"]
NAVIDROME_PASS = os.environ["NAVIDROME_PASS"]
MUSIC_FOLDER = os.environ.get("MUSIC_FOLDER", "/srv/media/music").rstrip("/")


def subsonic_call(endpoint: str, **params) -> dict:
    salt = secrets.token_hex(8)
    token = hashlib.md5((NAVIDROME_PASS + salt).encode()).hexdigest()
    base = {
        "u": NAVIDROME_USER,
        "t": token,
        "s": salt,
        "v": "1.16.1",
        "c": "streamcloud-troi",
        "f": "json",
    }
    qs = urllib.parse.urlencode({**base, **params})
    url = f"{NAVIDROME_URL}/rest/{endpoint}.view?{qs}"
    with urllib.request.urlopen(url, timeout=10) as r:
        return json.loads(r.read())["subsonic-response"]


def find_song(artist: str, title: str) -> str | None:
    """Return absolute file path for first matching song, or None."""
    query = f"{artist} {title}"
    resp = subsonic_call("search3", query=query, songCount=1, artistCount=0, albumCount=0)
    songs = resp.get("searchResult3", {}).get("song", [])
    if not songs:
        return None
    path = songs[0].get("path")
    if not path:
        return None
    return f"{MUSIC_FOLDER}/{path}"


def main():
    data = json.load(sys.stdin if len(sys.argv) < 2 else open(sys.argv[1]))
    tracks = data.get("playlist", {}).get("track", [])

    print("#EXTM3U")
    found = 0
    for t in tracks:
        artist = t.get("creator", "")
        title = t.get("title", "")
        if not artist or not title:
            continue
        path = find_song(artist, title)
        if path:
            print(f"#EXTINF:-1,{artist} - {title}")
            print(path)
            found += 1
        else:
            print(f"# not in library: {artist} - {title}", file=sys.stderr)

    print(f"matched {found}/{len(tracks)} tracks", file=sys.stderr)


if __name__ == "__main__":
    main()

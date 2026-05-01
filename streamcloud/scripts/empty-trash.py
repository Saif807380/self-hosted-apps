#!/usr/bin/env python3
import hashlib
import json
import os
import secrets
import sys
import urllib.parse
import urllib.request
from pathlib import Path

# --- CONFIGURATION ---
SCRIPT_DIR = Path(__file__).parent
PROJECT_ROOT = SCRIPT_DIR.parent
MUSIC_FOLDER = "/srv/media/music"


# Load environment variables
def load_env():
    env_file = PROJECT_ROOT / ".env"
    if env_file.exists():
        with open(env_file) as f:
            for line in f:
                if "=" in line and not line.startswith("#"):
                    k, v = line.strip().split("=", 1)
                    os.environ[k] = v


load_env()

NAVIDROME_URL = os.environ.get("NAVIDROME_URL", "http://127.0.0.1:4533").rstrip("/")
NAVIDROME_USER = os.environ.get("NAVIDROME_USER")
NAVIDROME_PASS = os.environ.get("NAVIDROME_PASS", "")

if not NAVIDROME_USER or not NAVIDROME_PASS:
    print("Error: NAVIDROME_USER or NAVIDROME_PASS not found in .env")
    sys.exit(1)


def subsonic_call(endpoint: str, **params) -> dict:
    salt = secrets.token_hex(8)
    token = hashlib.md5((NAVIDROME_PASS + salt).encode()).hexdigest()
    base = {
        "u": NAVIDROME_USER,
        "t": token,
        "s": salt,
        "v": "1.16.1",
        "c": "streamcloud-trash",
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


def navidrome_native_call(method: str, endpoint: str, data: dict = {}) -> dict:
    """Helper for Navidrome's internal/native API."""
    # 1. Login to get JWT
    login_url = f"{NAVIDROME_URL}/auth/login"
    login_data = json.dumps(
        {"username": NAVIDROME_USER, "password": NAVIDROME_PASS}
    ).encode()
    req = urllib.request.Request(
        login_url,
        data=login_data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    try:
        with urllib.request.urlopen(req, timeout=10) as r:
            token = json.loads(r.read())["token"]
    except Exception as e:
        print(f"Native API Login Failed: {e}")
        return {}

    # 2. Call the endpoint
    url = f"{NAVIDROME_URL}{endpoint}"
    headers = {
        "x-nd-authorization": f"Bearer {token}",
        "x-nd-client-unique-id": "streamcloud-trash-script",
        "Content-Type": "application/json",
    }

    body = json.dumps(data).encode() if data else None
    req = urllib.request.Request(url, data=body, headers=headers, method=method)

    try:
        with urllib.request.urlopen(req, timeout=10) as r:
            return json.loads(r.read())
    except Exception as e:
        print(f"Native API {method} {endpoint} Failed: {e}")
        return {}


def main():
    print("Checking for 'Trash' playlist...")
    resp = subsonic_call("getPlaylists")
    playlists = resp.get("playlists", {}).get("playlist", [])

    trash_id = None
    for pl in playlists:
        if pl.get("name", "").lower() == "trash":
            trash_id = pl.get("id")
            break

    if not trash_id:
        print("No playlist named 'Trash' found. Nothing to delete.")
        return

    print(f"Found 'Trash' playlist (ID: {trash_id}). Fetching tracks...")
    resp = subsonic_call("getPlaylist", id=trash_id)
    tracks = resp.get("playlist", {}).get("entry", [])

    if not tracks:
        print("Trash playlist is empty.")
        return

    print(f"Found {len(tracks)} tracks to delete.")
    deleted_count = 0

    for t in tracks:
        rel_path = t.get("path")
        if not rel_path:
            continue

        abs_path = Path(MUSIC_FOLDER) / rel_path
        if abs_path.exists():
            print(f"Deleting: {rel_path}")
            try:
                abs_path.unlink()
                deleted_count += 1
            except Exception as e:
                print(f"Error deleting {rel_path}: {e}")
        else:
            print(f"File not found on disk: {rel_path}")

    # Clean up empty directories
    print("Cleaning up empty directories...")
    import subprocess

    subprocess.run(["find", MUSIC_FOLDER, "-type", "d", "-empty", "-delete"])

    if deleted_count > 0:
        print(f"Successfully deleted {deleted_count} tracks.")

        print("Triggering Navidrome rescan...")
        subsonic_call("startScan")

        print("Triggering Navidrome 'Remove Missing Files' cleanup...")
        navidrome_native_call("DELETE", "/api/missing")

        print("🚀 Done! Files are gone and database has been cleaned up.")
    else:
        print("No files were deleted.")


if __name__ == "__main__":
    main()

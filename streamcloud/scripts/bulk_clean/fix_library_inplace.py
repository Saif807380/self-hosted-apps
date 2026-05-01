#!/usr/bin/env python3
from pathlib import Path

from mutagen.oggopus import OggOpus

# --- CONFIGURATION ---
TARGET_DIR = Path("/srv/media/music")


def fix_metadata_from_path():
    """
    Recursively scans the target directory and applies metadata based on the folder structure:
    /srv/media/music/Artist Name/Album Name/Track Title.opus
    """
    print(f"🚀 Starting in-place metadata sync from folder structure: {TARGET_DIR}")

    success_count = 0
    error_count = 0
    skipped_count = 0

    # Scan for all .opus files
    for p in TARGET_DIR.rglob("*.opus"):
        # We expect at least: TARGET_DIR / Artist / Album / Track.opus
        # So the path parts relative to TARGET_DIR should have length 3
        rel_parts = p.relative_to(TARGET_DIR).parts

        if len(rel_parts) < 3:
            print(f"⚠️  Skipping (Path too shallow): {p.relative_to(TARGET_DIR)}")
            skipped_count += 1
            continue

        # Path breakdown:
        # rel_parts[0] = Artist
        # rel_parts[1] = Album
        # p.stem = Track Title (filename without extension)

        artist = rel_parts[0]
        album = rel_parts[1]
        title = p.stem

        try:
            audio = OggOpus(p)

            # Update tags
            audio["artist"] = [artist]
            audio["albumartist"] = [artist]
            audio["album"] = [album]
            audio["title"] = [title]

            audio.save()
            success_count += 1
            if success_count % 100 == 0:
                print(f"Progress: Updated {success_count} tracks...")

        except Exception as e:
            print(f"❌ Error processing {p}: {e}")
            error_count += 1

    print("\n" + "=" * 30)
    print("✅ SYNC COMPLETE")
    print("=" * 30)
    print(f"Updated: {success_count}")
    print(f"Skipped: {skipped_count}")
    print(f"Errors:  {error_count}")
    print("=" * 30)


if __name__ == "__main__":
    if not TARGET_DIR.exists():
        print(f"Error: Target directory {TARGET_DIR} does not exist.")
    else:
        fix_metadata_from_path()

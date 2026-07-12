#!/usr/bin/env python3
import json
import os
from pathlib import Path
from mutagen.oggopus import OggOpus

# --- CONFIGURATION ---
INPUT_DIR = Path(os.environ.get("INPUT_DIR", os.path.join(os.path.dirname(__file__), "../llm_inputs")))
OUTPUT_DIR = Path(os.environ.get("OUTPUT_DIR", os.path.join(os.path.dirname(__file__), "../llm_outputs")))


def real_paths_by_index(output_file, n_items):
    """Return the authoritative source paths from the matching input batch.

    The LLM often normalizes filenames (NFC vs NFD, fullwidth chars), so its
    echoed "path" may not match the file on disk. The input batch holds the real
    paths straight from clean_lib's scan; pair by index. Fall back to the echoed
    path only if the input batch is missing or its length differs.
    """
    input_file = INPUT_DIR / output_file.name
    if not input_file.exists():
        return None
    inp = json.loads(input_file.read_text())
    if len(inp) != n_items:
        print(f"⚠️  {output_file.name}: input/output length mismatch "
              f"({len(inp)} vs {n_items}); using LLM-echoed paths for this batch.")
        return None
    return [i.get("path") for i in inp]


def apply_metadata(mapping_file):
    print(f"Applying metadata from {mapping_file.name}...")
    try:
        with open(mapping_file, "r") as f:
            mapping = json.load(f)
    except json.JSONDecodeError:
        print(f"Error: {mapping_file.name} contains invalid JSON. Skipping.")
        return

    real_paths = real_paths_by_index(mapping_file, len(mapping))

    for idx, item in enumerate(mapping):
        file_path = real_paths[idx] if real_paths else item.get("path")
        if not file_path or not os.path.exists(file_path):
            print(f"Skipping (File Not Found): {file_path}")
            continue

        try:
            audio = OggOpus(file_path)
            
            # Apply tags from LLM
            audio['title'] = item.get('title', '')
            audio['album'] = item.get('album', '')
            audio['artist'] = item.get('artist', '')
            audio['albumartist'] = item.get('albumartist', '')
            
            audio.save()
            print(f"✅ Tagged: {os.path.basename(file_path)} -> {item.get('artist')} / {item.get('title')}")
        except Exception as e:
            print(f"❌ Failed to tag {file_path}: {e}")

def main():
    if not OUTPUT_DIR.exists():
        print(f"Error: The directory '{OUTPUT_DIR.resolve()}' does not exist.")
        return

    batch_files = list(OUTPUT_DIR.glob("*.json"))
    if not batch_files:
        print(f"No JSON files found in {OUTPUT_DIR.resolve()}")
        return

    for batch_file in sorted(batch_files):
        apply_metadata(batch_file)

if __name__ == "__main__":
    main()

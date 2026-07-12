import json
import os
import shutil
from pathlib import Path

# --- CONFIGURATION ---
TARGET_DIR = Path(os.environ.get("TARGET_DIR", "/srv/media/music"))
INPUT_DIR = Path(os.environ.get("INPUT_DIR", os.path.join(os.path.dirname(__file__), "../llm_inputs")))
OUTPUT_DIR = Path(os.environ.get("OUTPUT_DIR", os.path.join(os.path.dirname(__file__), "../llm_outputs")))


def real_paths_by_index(output_file, n_items):
    """Authoritative source paths from the matching input batch, paired by index.

    The LLM often normalizes filenames (NFC vs NFD, fullwidth chars) in its echoed
    "path", so it won't match the file on disk. The input batch holds the real
    paths from clean_lib's scan. Fall back to the echoed path only if the input
    batch is missing or its length differs.
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


def main():
    if not OUTPUT_DIR.exists():
        print(f"Error: The directory '{OUTPUT_DIR.resolve()}' does not exist.")
        return

    batch_files = list(OUTPUT_DIR.glob("*.json"))
    if not batch_files:
        print(f"No JSON files found in {OUTPUT_DIR.resolve()}")
        return

    total_moved = 0
    total_skipped = 0
    total_replaced = 0

    print("Moving files to standardized structure...")

    for batch_file in sorted(batch_files):
        try:
            with open(batch_file, "r") as f:
                mapping = json.load(f)
        except json.JSONDecodeError:
            print(f"Error: {batch_file.name} contains invalid JSON. Skipping.")
            continue

        real_paths = real_paths_by_index(batch_file, len(mapping))

        for idx, item in enumerate(mapping):
            src_path_str = real_paths[idx] if real_paths else item.get("path")
            new_rel_path = item.get("new_path")

            if not src_path_str or not new_rel_path:
                continue

            src_path = Path(src_path_str)
            # Remove leading slash from new_rel_path if present to join correctly
            new_rel_path = new_rel_path.lstrip("/")
            
            dst_path = TARGET_DIR / new_rel_path

            if not src_path.exists():
                print(f"Skipping (Not Found in Source): {src_path.name}")
                total_skipped += 1
                continue

            # If the destination already exists, we perform a replace operation
            if dst_path.exists():
                print(f"🔄 Replacing existing file: {dst_path}")
                try:
                    dst_path.unlink()
                    total_replaced += 1
                except Exception as e:
                    print(f"❌ Failed to remove existing file {dst_path}: {e}")
                    total_skipped += 1
                    continue

            # Create directories and move
            dst_path.parent.mkdir(parents=True, exist_ok=True)

            try:
                shutil.move(str(src_path), str(dst_path))
                total_moved += 1
            except Exception as e:
                print(f"❌ Failed to move {src_path.name}: {e}")
                total_skipped += 1

    print("\n" + "=" * 30)
    print("🚀 MOVEMENT COMPLETE")
    print("=" * 30)
    print(f"Successfully Moved: {total_moved}")
    print(f"Files Replaced:     {total_replaced}")
    print(f"Skipped/Errors:     {total_skipped}")
    print("=" * 30)


if __name__ == "__main__":
    main()

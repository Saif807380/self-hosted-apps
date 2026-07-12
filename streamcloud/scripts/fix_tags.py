#!/usr/bin/env python3
import json
import shutil
import subprocess
import sys
from pathlib import Path

# Paths relative to this script
SCRIPT_DIR = Path(__file__).parent
BULK_CLEAN_DIR = SCRIPT_DIR / "bulk_clean"
PROMPT_FILE = BULK_CLEAN_DIR / "clean_up_prompt.md"
INPUT_DIR = SCRIPT_DIR / "llm_inputs"
OUTPUT_DIR = SCRIPT_DIR / "llm_outputs"

MODEL = "haiku"
CLAUDE_BIN = shutil.which("claude") or "/usr/bin/claude"
REQUIRED_KEYS = {"path", "title", "album", "artist", "albumartist", "new_path"}


def extract_json_array(text):
    """Extract and validate a JSON array from LLM stdout.

    Uses raw_decode to consume the JSON array from its opening '[' and stop
    immediately, ignoring any trailing prose (Sources section, fences, etc.).
    """
    start = text.find("[")
    if start == -1:
        raise ValueError("No JSON array found in output")

    # raw_decode consumes exactly the JSON value and returns the rest index,
    # so trailing markdown fences, Sources lists, etc. are all ignored.
    decoder = json.JSONDecoder()
    items, _ = decoder.raw_decode(text, start)

    if not isinstance(items, list):
        raise ValueError("Parsed JSON is not a list")
    for item in items:
        missing = REQUIRED_KEYS - item.keys()
        if missing:
            raise ValueError(f"Item missing keys: {missing} — {item.get('path', '?')}")

    return items


def run_batch(prompt_template, batch_path):
    """Run one claude call for a single batch file. Returns parsed items or raises."""
    batch_text = batch_path.read_text()
    prompt = f"{prompt_template}\n\nInput array:\n{batch_text}"

    result = subprocess.run(
        [CLAUDE_BIN, "-p", prompt,
         "--model", MODEL,
         "--allowedTools", "WebSearch", "WebFetch",
         "--output-format", "text"],
        capture_output=True,
        text=True,
        stdin=subprocess.DEVNULL,
        timeout=600,
    )

    if result.returncode != 0:
        raise RuntimeError(
            f"claude exited {result.returncode}:\n{result.stderr.strip()}"
        )

    return extract_json_array(result.stdout)


def process_batches(prompt_template):
    """Process every batch in INPUT_DIR, writing validated JSON to OUTPUT_DIR.

    Retries each batch once on failure; logs a warning and continues on second failure
    so one bad batch never kills the whole run.
    """
    batch_files = sorted(INPUT_DIR.glob("batch_*.json"))
    succeeded = 0

    for batch_path in batch_files:
        print(f"  Processing {batch_path.name} …", flush=True)
        for attempt in (1, 2):
            try:
                items = run_batch(prompt_template, batch_path)
                out_path = OUTPUT_DIR / batch_path.name
                with open(out_path, "w") as f:
                    json.dump(items, f, indent=2)
                print(f"    ✓ {batch_path.name}: {len(items)} tracks written", flush=True)
                succeeded += 1
                break
            except Exception as e:
                if attempt == 1:
                    print(f"    Attempt 1 failed ({e}). Retrying …", flush=True)
                else:
                    print(f"    ✗ {batch_path.name}: skipped after 2 failures — {e}", flush=True)

    return succeeded


def run_script(script_name, *args):
    script_path = BULK_CLEAN_DIR / script_name
    cmd = [sys.executable, str(script_path)] + list(args)
    subprocess.run(cmd, check=True)


def main():
    print("--- Pipeline: Stage 1 (Preparation) ---")
    INPUT_DIR.mkdir(exist_ok=True)
    for f in INPUT_DIR.glob("*.json"):
        f.unlink()
    OUTPUT_DIR.mkdir(exist_ok=True)
    for f in OUTPUT_DIR.glob("*.json"):
        f.unlink()

    run_script("clean_lib.py")

    if not list(INPUT_DIR.glob("*.json")):
        print("No files to process. Exiting.")
        return

    print("\n--- Pipeline: Stage 2 (LLM Metadata Correction) ---")
    prompt_template = PROMPT_FILE.read_text()
    succeeded = process_batches(prompt_template)

    if succeeded == 0:
        print("Error: all batches failed. Aborting before tagging/moving.")
        sys.exit(1)

    print(f"\n--- Pipeline: Stage 3 (Tagging & Moving) ---")
    run_script("clean_tags.py")
    run_script("move_files.py")

    print("\n🚀 Pipeline complete! Files have been tagged and moved to /srv/media/music")


if __name__ == "__main__":
    main()

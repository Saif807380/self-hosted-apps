import json
import os
from pathlib import Path

# --- CONFIGURATION ---
SOURCE_DIR = os.environ.get("SOURCE_DIR", os.path.expanduser("~/Music"))
INPUT_DIR = os.environ.get("INPUT_DIR", os.path.join(os.path.dirname(__file__), "../llm_inputs"))
BATCH_SIZE = int(os.environ.get("BATCH_SIZE", 200))
AUDIO_EXTENSIONS = {".opus"}


def main():
    source_path = Path(SOURCE_DIR)
    input_path = Path(INPUT_DIR)

    if not source_path.exists():
        print(f"Error: Directory '{source_path.resolve()}' does not exist.")
        return

    input_path.mkdir(parents=True, exist_ok=True)

    files = []
    for file_path in source_path.rglob("*"):
        # Skip yt-dlp postprocessing leftovers: "X.temp.opus" has suffix ".opus"
        # and would otherwise be batched as a (0-byte, unparseable) real track.
        if ".temp." in file_path.name or file_path.name.endswith(".part"):
            continue
        if file_path.is_file() and file_path.suffix.lower() in AUDIO_EXTENSIONS:
            # We use absolute path so the subsequent tagging script knows exactly where the file is
            files.append({
                "path": str(file_path.absolute()),
                "filename": file_path.name
            })

    total_files = len(files)
    if total_files == 0:
        print("No audio files found.")
        return

    # Chunk the files and write to batch files
    for i in range(0, total_files, BATCH_SIZE):
        batch = files[i : i + BATCH_SIZE]
        batch_number = (i // BATCH_SIZE) + 1

        batch_filename = input_path / f"batch_{batch_number:03d}.json"

        with open(batch_filename, "w") as f:
            json.dump(batch, f, indent=2)

        print(f"Created {batch_filename.name} ({len(batch)} tracks)")

    print(
        f"\nTotal: Exported {total_files} tracks into {total_files // BATCH_SIZE + (1 if total_files % BATCH_SIZE != 0 else 0)} batch files."
    )


if __name__ == "__main__":
    main()

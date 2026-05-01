import json
from pathlib import Path

# --- CONFIGURATION ---
INPUT_DIR = Path("../llm_inputs")
OUTPUT_DIR = Path("../llm_outputs")


def get_paths_from_dir(directory):
    """Reads all JSON files in a directory and extracts the 'current_path' values."""
    paths = set()

    if not directory.exists():
        print(f"Warning: Directory {directory} does not exist.")
        return paths

    for file_path in directory.glob("*.json"):
        try:
            with open(file_path, "r", encoding="utf-8") as f:
                data = json.load(f)

                # Ensure data is a list
                if not isinstance(data, list):
                    print(
                        f"Warning: {file_path.name} does not contain a JSON array. Skipping."
                    )
                    continue

                for item in data:
                    if isinstance(item, dict) and "current_path" in item:
                        paths.add(item["current_path"])
        except json.JSONDecodeError:
            print(f"Error: {file_path.name} contains invalid JSON.")
        except Exception as e:
            print(f"Error reading {file_path.name}: {e}")

    return paths


def main():
    print("Gathering input paths...")
    input_paths = get_paths_from_dir(INPUT_DIR)

    print("Gathering output paths...")
    output_paths = get_paths_from_dir(OUTPUT_DIR)

    # Calculate differences
    missing_paths = input_paths - output_paths
    hallucinated_paths = output_paths - input_paths

    # Print Summary
    print("\n" + "=" * 30)
    print("🔍 RECONCILIATION SUMMARY")
    print("=" * 30)
    print(f"Total Input Tracks Expected: {len(input_paths)}")
    print(f"Total Output Tracks Found:   {len(output_paths)}")
    print("-" * 30)
    print(f"❌ Missing Tracks:      {len(missing_paths)}")
    print(
        f"⚠️ Hallucinated Tracks: {len(hallucinated_paths)} (Paths LLM invented or altered)"
    )
    print("=" * 30)

    # Print Details
    if missing_paths:
        print("\n--- MISSING TRACKS (Failed to process) ---")
        for path in sorted(missing_paths):
            print(path)

    if hallucinated_paths:
        print("\n--- HALLUCINATED TRACKS (In output, but not in input) ---")
        for path in sorted(hallucinated_paths):
            print(path)

    if not missing_paths and not hallucinated_paths:
        print("\n✅ Perfect match! All files were successfully mapped.")


if __name__ == "__main__":
    main()

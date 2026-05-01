#!/usr/bin/env python3
import os
import sys
import subprocess
import tempfile
from pathlib import Path

# Paths relative to this script
SCRIPT_DIR = Path(__file__).parent
BULK_CLEAN_DIR = SCRIPT_DIR / "bulk_clean"
PROMPT_FILE = BULK_CLEAN_DIR / "clean_up_prompt.md"
INPUT_DIR = SCRIPT_DIR / "llm_inputs"
OUTPUT_DIR = SCRIPT_DIR / "llm_outputs"

# Shortlisted tools required for the metadata correction prompt
REQUIRED_TOOLS = [
    "google_web_search",
    "web_fetch",
    "read_file",
    "list_directory",
    "grep_search",
    "glob"
]

def run_script(script_name, *args):
    script_path = BULK_CLEAN_DIR / script_name
    cmd = [sys.executable, str(script_path)] + list(args)
    subprocess.run(cmd, check=True)

def main():
    print("--- Pipeline: Stage 1 (Preparation) ---")
    # Ensure directories exist and are clean
    INPUT_DIR.mkdir(exist_ok=True)
    for f in INPUT_DIR.glob("*.json"): 
        f.unlink()
    OUTPUT_DIR.mkdir(exist_ok=True)
    for f in OUTPUT_DIR.glob("*.json"): 
        f.unlink()
    
    # Step 1: Generate batches
    run_script("clean_lib.py")
    
    # Check if we have anything to process
    if not list(INPUT_DIR.glob("*.json")):
        print("No files to process. Exiting.")
        return

    print("\n--- Pipeline: Stage 2 (LLM Metadata Correction) ---")
    
    with open(PROMPT_FILE, "r") as f:
        prompt_content = f.read()
    
    full_prompt = f"{prompt_content}\n\nInputs are in: {INPUT_DIR.absolute()}\nOutputs should go to: {OUTPUT_DIR.absolute()}"
    
    # Create a temporary policy file to allow required tools
    with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as temp_policy:
        policy_content = "allow:\n" + "\n".join([f"  - tool: {tool}" for tool in REQUIRED_TOOLS])
        temp_policy.write(policy_content)
        policy_path = temp_policy.name

    print(f"Invoking Gemini with auto-approval for: {', '.join(REQUIRED_TOOLS)} and filesystem edits...")
    
    # --approval-mode auto_edit handles write_file/replace
    # --policy handles our surgical list of read/search tools
    cmd = [
        "gemini", 
        "--model", "gemini-3-flash-preview",
        "--approval-mode", "auto_edit", 
        "--policy", policy_path, 
        "--prompt", full_prompt
    ]
    
    try:
        subprocess.run(cmd, check=True)
    except subprocess.CalledProcessError as e:
        print(f"Error: Gemini CLI failed with exit code {e.returncode}")
        sys.exit(1)
    finally:
        if os.path.exists(policy_path):
            os.unlink(policy_path)

    print("\n--- Pipeline: Stage 3 (Tagging & Moving) ---")
    # Step 3: Apply tags and move files
    run_script("clean_tags.py")
    run_script("move_files.py")
    
    print("\n🚀 Pipeline complete! Files have been tagged and moved to /srv/media/music")

if __name__ == "__main__":
    main()

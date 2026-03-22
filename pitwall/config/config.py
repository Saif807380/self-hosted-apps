import os
import sys
from dotenv import load_dotenv

load_dotenv()

def _require(var: str) -> str:
    val = os.getenv(var)
    if not val:
        print(f"Error: {var} is required. Set it in .env or environment.", file=sys.stderr)
        sys.exit(1)
    return val

# API keys
GEMINI_API_KEY = _require("GEMINI_API_KEY")
OWM_API_KEY = os.getenv("OWM_API_KEY", "")

# Configurable
GEMINI_MODEL = os.getenv("GEMINI_MODEL", "gemini-2.0-flash")
RACE_YEAR = int(os.getenv("RACE_YEAR", "2026"))
RACE_ROUND = int(os.getenv("RACE_ROUND", "1"))
CIRCUIT_CITY = os.getenv("CIRCUIT_CITY", "Melbourne")
CIRCUIT_COUNTRY_CODE = os.getenv("CIRCUIT_COUNTRY_CODE", "AU")
IS_SPRINT_WEEKEND = os.getenv("IS_SPRINT_WEEKEND", "false").lower() == "true"

# Game rules
MAX_DRIVERS = 5
MAX_CONSTRUCTORS = 2
SWAP_PENALTY_POINTS = -10  # per transfer beyond free allowance
TOTAL_BUDGET = float(os.getenv("TOTAL_BUDGET", "100.0"))

# Rules file path (fed to the LLM alongside data)
RULES_FILE = os.path.join(os.path.dirname(__file__), "config", "Rules.md")

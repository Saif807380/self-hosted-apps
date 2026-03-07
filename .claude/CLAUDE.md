## Development Philosophy
* Simplicity: Write simple, straightforward code
* Readability: Make code easy to understand
* Performance: Consider performance without sacrificing readability
* Maintainability: Write code that's easy to update
* Testability: Ensure code is testable
* Reusability: Create reusable components and functions
* Less Code = Less Debt: Minimize code footprint
* Security: Put creds in an environment or properties file depending on the application and never commit them

## Coding Best Practices

* Early Returns: Use to avoid nested conditions
* Descriptive Names: Use clear variable/function names (prefix handlers with "handle")
* Constants Over Functions: Use constants where possible
* DRY Code: Don't repeat yourself
* Functional Style: Prefer functional, immutable approaches when not verbose
* Minimal Changes: Only modify code related to the task at hand
* Function Ordering: Define composing functions before their components
* TODO Comments: Mark issues in existing code with "TODO:" prefix
* Simplicity: Prioritize simplicity and readability over clever solutions
* Functional Code: Use functional and stateless approaches where they improve clarity
* Clean logic: Keep core logic clean and push implementation details to the edges
* File Organsiation: Balance file organization with simplicity - use an appropriate number of files for the project scale

## Instructions for implementing tasks

Always implement tasks one by one. When one task is finished, wait for my approval to start the next

## Pitwall — F1 Fantasy Optimizer

### Stack
- Python 3.14, venv, pip
- LangChain + LangGraph (ReAct agent), Gemini LLM (langchain-google-genai)
- fastf1 (telemetry), feedparser (RSS news), OpenWeatherMap (weather)
- rich (CLI output)

### Key Gotchas
- **Empty `__init__.py` files**: `telemetry/__init__.py` and `agent/__init__.py` MUST be empty — eager imports cause deadlocks when LangChain agent tools do lazy imports
- **fastf1 logging**: Set `logging.getLogger("fastf1").setLevel(logging.ERROR)` — extremely verbose by default
- **fastf1 NaT values**: Use `pd.isna()` not `is not None` — fastf1 uses pandas NaT, not Python None
- **fastf1 lap filtering**: Use `IsAccurate` flag, not manual pit/track-status checks
- **fastf1 lightweight loading**: For results-only queries use `session.load(laps=False, telemetry=False, weather=False, messages=False)`
- **fastf1 event search**: Search `Location` and `Country` columns, not just `EventName` (e.g., "Melbourne" is in Location, not EventName)
- **Gemini model names**: Use `gemini-2.5-flash` or `gemini-2.5-pro` — `3.x-preview` models require thought signatures unsupported by LangChain
- **Season data fallback**: When current year data unavailable in fastf1, fall back to previous year with a note
- **Scraper abandoned**: Fantasy state comes from manually maintained JSON files in `config/` (current_team.json, market.json), not from web scraping
- **OWM_API_KEY is optional**: Weather is nice-to-have, not critical — don't `sys.exit` if missing
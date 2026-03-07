# Pitwall

F1 Fantasy roster optimizer. Analyzes real telemetry, weather, and recent news to recommend the optimal driver/constructor lineup each race weekend.

## How it works

A LangChain ReAct agent (Gemini LLM) calls 6 tools in parallel, then reasons over the combined data to produce a roster recommendation:

1. **Fantasy state** — your current team and full market prices (from manually maintained JSON files)
2. **Practice telemetry** — FP1/FP2 long-run pace and consistency via fastf1
3. **Season form** — aggregated race and qualifying results, points trend, reliability
4. **Circuit history** — past 3 years of results at the current circuit
5. **Weather forecast** — qualifying and race day conditions via OpenWeatherMap
6. **Recent news** — grid penalties, injuries, upgrades, DNS risks via F1 RSS feeds

The agent applies weighted criteria (see `agent/prompts.py`) and outputs a Markdown roster table with budget arithmetic and swap justification.

## Setup

```bash
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

Copy `.env.example` to `.env` and fill in your credentials:

```bash
cp .env.example .env
```

| Variable | Required | Description |
|---|---|---|
| `GEMINI_API_KEY` | Yes | Google AI Studio API key |
| `GEMINI_MODEL` | No | Model to use (default: `gemini-2.5-flash`) |
| `OWM_API_KEY` | No | OpenWeatherMap API key for weather forecasts |
| `RACE_YEAR` | No | Season year (default: 2026) |
| `RACE_ROUND` | No | Round number (default: 1) |
| `CIRCUIT_CITY` | No | Circuit city for weather/history (default: Melbourne) |
| `CIRCUIT_COUNTRY_CODE` | No | ISO country code for weather (default: AU) |

## Fantasy data

Pitwall reads your current team and the full market from manually maintained JSON files. Update these before each race:

- `config/current_team.json` — your active roster, budget, and available swaps
- `config/market.json` — all drivers and constructors with current prices

## Usage

```bash
# Full run (uses live telemetry and LLM)
python main.py

# Override race round
python main.py --year 2026 --round 3

# Show agent intermediate reasoning
python main.py --verbose

# Dry run (fixture data, no live API calls)
python main.py --dry-run
```

## Testing modules

Test individual data sources without invoking the LLM:

```bash
# Practice pace
python test_modules.py practice --year 2025 --round 2

# Qualifying results
python test_modules.py quali --year 2025 --round 2

# Race results
python test_modules.py race --year 2025 --round 2

# Season summary
python test_modules.py season --year 2025 --round 5

# Circuit history
python test_modules.py history

# Weather forecast
python test_modules.py weather

# Recent news
python test_modules.py news
```

## Directory structure

```
pitwall/
├── config/               Manually maintained race data
│   ├── current_team.json Current roster, budget, available swaps
│   ├── market.json       All drivers and constructors with prices
│   └── Rules.md          F1 Fantasy scoring rules (fed to LLM)
├── agent/
│   ├── tools.py          LangChain @tool wrappers for each data source
│   ├── prompts.py        System prompt and CRITERIA dict (tweak weights here)
│   ├── orchestrator.py   ReAct agent setup and execution
│   └── dry_run_tools.py  Fixture-backed tools for --dry-run mode
├── telemetry/
│   ├── session_data.py   fastf1: practice pace, qualifying, race, season summary
│   ├── historical.py     fastf1: circuit history across multiple years
│   ├── weather.py        OpenWeatherMap: qualifying and race day forecast
│   └── news.py           RSS: recent F1 headlines from multiple sources
├── tests/fixtures/       Sample JSON for dry-run mode
├── main.py               CLI entrypoint
├── config.py             Environment variable loading
└── requirements.txt
```

## Adjusting criteria weights

Edit `CRITERIA` in `agent/prompts.py` to tune the LLM's prioritization between races:

```python
CRITERIA = {
    "practice_pace":     {"weight": "HIGH", ...},
    "price_value":       {"weight": "HIGH", ...},
    "season_form":       {"weight": "HIGH", ...},
    "news_context":      {"weight": "MEDIUM", ...},
    "weather_chaos":     {"weight": "MEDIUM", ...},
    "historical_circuit":{"weight": "LOW", ...},
}
```

No other code changes needed.

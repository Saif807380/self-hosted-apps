from pathlib import Path

from langchain_core.tools import tool

FIXTURES_DIR = Path(__file__).resolve().parent.parent / "tests" / "fixtures"


def _load_fixture(name: str) -> str:
    return (FIXTURES_DIR / name).read_text()


@tool
def get_fantasy_state() -> str:
    """Get the user's current F1 Fantasy state: budget, available swaps, current roster (drivers + constructors), and power-ups. (DRY RUN — fixture data)"""
    return _load_fixture("fantasy_state.json")


@tool
def get_practice_telemetry() -> str:
    """Get FP1/FP2 long-run pace analysis for all drivers at the current race weekend. (DRY RUN — fixture data)"""
    return _load_fixture("practice_pace.json")


@tool
def get_season_form() -> str:
    """Get aggregated season results up to the current round. (DRY RUN — fixture data)"""
    return _load_fixture("season_summary.json")


@tool
def get_circuit_history_data() -> str:
    """Get historical race results at the current circuit for the past 3 years. (DRY RUN — fixture data)"""
    return _load_fixture("circuit_history.json")


@tool
def get_weather_forecast() -> str:
    """Get the weather forecast for qualifying and race day. (DRY RUN — fixture data)"""
    return _load_fixture("weather.json")


@tool
def get_recent_news() -> str:
    """Get recent F1 news headlines and summaries. (DRY RUN — fixture data)"""
    return _load_fixture("news.json")


ALL_DRY_RUN_TOOLS = [
    get_fantasy_state,
    get_practice_telemetry,
    get_season_form,
    get_circuit_history_data,
    get_weather_forecast,
    get_recent_news,
]

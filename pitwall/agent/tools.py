import json
from dataclasses import asdict
from pathlib import Path

from langchain_core.tools import tool

import pitwall.config.config as config

CONFIG_DIR = Path(__file__).resolve().parent.parent / "config"


@tool
def get_fantasy_state() -> str:
    """Get the user's current F1 Fantasy roster (drivers, constructors, points, turbo driver) and the full market with pricing for all available drivers and constructors."""
    try:
        current_team = json.loads((CONFIG_DIR / "current_team.json").read_text())
        market = json.loads((CONFIG_DIR / "market.json").read_text())
        return json.dumps({
            "current_team": current_team,
            "market": market["market"],
        }, indent=2)
    except Exception as e:
        return json.dumps({"error": f"Failed to load fantasy data: {e}"})


@tool
def get_practice_telemetry() -> str:
    """Get FP1/FP2 long-run pace analysis for all drivers at the current race weekend. Returns avg pace, lap count, and consistency (lower stdev = more consistent). Falls back to previous year's same round if current year data is not available."""
    try:
        from telemetry.session_data import get_practice_pace
        paces = get_practice_pace(config.RACE_YEAR, config.RACE_ROUND)
        if paces:
            return json.dumps({"year": config.RACE_YEAR, "round": config.RACE_ROUND, "data": [asdict(p) for p in paces]}, indent=2)

        # Fall back to previous year's same round
        prev_year = config.RACE_YEAR - 1
        paces = get_practice_pace(prev_year, config.RACE_ROUND)
        if paces:
            return json.dumps({"year": prev_year, "round": config.RACE_ROUND, "note": f"Using {prev_year} data as {config.RACE_YEAR} practice data is not yet available. Driver lineups may differ.", "data": [asdict(p) for p in paces]}, indent=2)

        return json.dumps({"info": f"No practice data available for {config.RACE_YEAR} or {prev_year} R{config.RACE_ROUND}."})
    except Exception as e:
        return json.dumps({"error": f"Failed to load practice data: {e}"})


@tool
def get_season_form() -> str:
    """Get aggregated season results: per-driver total points, avg qualifying/finish position, DNF count, recent points trend; per-constructor total points and reliability. Falls back to previous year if current season hasn't started."""
    try:
        from telemetry.session_data import get_season_summary
        summary = get_season_summary(config.RACE_YEAR, config.RACE_ROUND)
        if summary.rounds_completed > 0:
            return json.dumps({"year": config.RACE_YEAR, "data": asdict(summary)}, indent=2)

        # Fall back to previous year's full season
        prev_year = config.RACE_YEAR - 1
        # Use a high round number to get the full previous season
        summary = get_season_summary(prev_year, 25)
        if summary.rounds_completed > 0:
            return json.dumps({"year": prev_year, "note": f"Using {prev_year} full season data as {config.RACE_YEAR} season has not started yet. Driver lineups may differ.", "data": asdict(summary)}, indent=2)

        return json.dumps({"info": f"No season data available for {config.RACE_YEAR} or {prev_year}."})
    except Exception as e:
        return json.dumps({"error": f"Failed to load season summary: {e}"})


@tool
def get_circuit_history_data() -> str:
    """Get historical race results at the current circuit for the past 3 years. Shows which drivers/teams tend to perform well here."""
    try:
        from telemetry.historical import get_circuit_history
        history = get_circuit_history(config.CIRCUIT_CITY, years=3)
        if not history:
            return json.dumps({"info": f"No historical data found for circuit '{config.CIRCUIT_CITY}' in the last 3 years."})
        return json.dumps(history, indent=2)
    except Exception as e:
        return json.dumps({"error": f"Failed to load circuit history for {config.CIRCUIT_CITY}: {e}"})


@tool
def get_weather_forecast() -> str:
    """Get the weather forecast for qualifying (Saturday) and race day (Sunday): temperature, conditions, rain probability, wind speed."""
    try:
        from telemetry.weather import get_race_weekend_weather
        forecast = get_race_weekend_weather(config.CIRCUIT_CITY, config.CIRCUIT_COUNTRY_CODE)
        return json.dumps({
            "qualifying_day": asdict(forecast.qualifying_day) if forecast.qualifying_day else None,
            "race_day": asdict(forecast.race_day) if forecast.race_day else None,
        }, indent=2)
    except Exception as e:
        return json.dumps({"error": f"Failed to load weather for {config.CIRCUIT_CITY}: {e}"})


@tool
def get_recent_news() -> str:
    """Get recent F1 news headlines and summaries from the past 5 days. Useful for identifying grid penalties, driver injuries, car upgrades/damage, team orders, and DNS risks."""
    try:
        from telemetry.news import get_f1_news
        news = get_f1_news(days_back=5)
        if not news:
            return json.dumps({"info": "No recent F1 news found in the last 5 days."})
        return json.dumps([asdict(n) for n in news], indent=2, default=str)
    except Exception as e:
        return json.dumps({"error": f"Failed to fetch F1 news: {e}"})


ALL_TOOLS = [
    get_fantasy_state,
    get_practice_telemetry,
    get_season_form,
    get_circuit_history_data,
    get_weather_forecast,
    get_recent_news,
]

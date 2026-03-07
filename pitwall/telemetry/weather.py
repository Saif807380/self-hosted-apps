from dataclasses import dataclass
from datetime import datetime, timedelta

import requests

import pitwall.config.config as config

OWM_FORECAST_URL = "https://api.openweathermap.org/data/2.5/forecast"


@dataclass
class DayForecast:
    date: str
    temp_celsius: float
    conditions: str
    rain_probability: float
    wind_speed_ms: float


@dataclass
class WeatherForecast:
    qualifying_day: DayForecast | None
    race_day: DayForecast | None


def _get_forecast_data(city: str, country_code: str) -> list[dict]:
    resp = requests.get(OWM_FORECAST_URL, params={
        "q": f"{city},{country_code}",
        "appid": config.OWM_API_KEY,
        "units": "metric",
    }, timeout=10)
    resp.raise_for_status()
    return resp.json().get("list", [])


def _find_day_forecast(forecasts: list[dict], target_date: str) -> DayForecast | None:
    day_entries = [f for f in forecasts if f["dt_txt"].startswith(target_date)]
    if not day_entries:
        return None

    # Pick the midday entry (closest to 14:00 local — race time is typically afternoon)
    midday = min(day_entries, key=lambda f: abs(int(f["dt_txt"].split(" ")[1].split(":")[0]) - 14))

    main = midday["main"]
    weather = midday["weather"][0] if midday.get("weather") else {}
    rain_prob = midday.get("pop", 0.0)

    return DayForecast(
        date=target_date,
        temp_celsius=main["temp"],
        conditions=weather.get("description", "unknown"),
        rain_probability=rain_prob,
        wind_speed_ms=midday.get("wind", {}).get("speed", 0.0),
    )


def get_race_weekend_weather(city: str, country_code: str) -> WeatherForecast:
    forecasts = _get_forecast_data(city, country_code)

    # Find upcoming Saturday (qualifying) and Sunday (race)
    today = datetime.now()
    days_until_saturday = (5 - today.weekday()) % 7
    if days_until_saturday == 0 and today.hour > 18:
        days_until_saturday = 7

    saturday = today + timedelta(days=days_until_saturday)
    sunday = saturday + timedelta(days=1)

    sat_str = saturday.strftime("%Y-%m-%d")
    sun_str = sunday.strftime("%Y-%m-%d")

    return WeatherForecast(
        qualifying_day=_find_day_forecast(forecasts, sat_str),
        race_day=_find_day_forecast(forecasts, sun_str),
    )

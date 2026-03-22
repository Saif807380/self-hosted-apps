"""Standalone test script for telemetry modules. No LLM calls."""

import argparse
import json
import sys
from pathlib import Path

# Allow running from either the pitwall/ dir or its parent
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

from dotenv import load_dotenv
from rich.console import Console
from rich.markdown import Markdown

load_dotenv()

console = Console()

def test_telemetry_practice(year: int, round_num: int):
    console.print("\n[bold cyan]--- Practice Pace (fastf1) ---[/bold cyan]\n")
    try:
        from pitwall.telemetry.session_data import get_practice_pace
        with console.status(f"Loading FP1/FP2 data for {year} R{round_num}..."):
            paces = get_practice_pace(year, round_num)

        console.print(f"[green]{len(paces)} drivers:[/green]")
        for p in paces:
            console.print(f"  {p.driver:4s} ({p.team:20s}) - {p.avg_long_run_pace:.3f}s avg, {p.lap_count} laps, stdev {p.consistency:.3f}")
        console.print("\n[bold green]Practice Pace: OK[/bold green]")
    except Exception as e:
        console.print(f"[bold red]Practice Pace: FAILED - {e}[/bold red]")


def test_telemetry_quali(year: int, round_num: int):
    console.print("\n[bold cyan]--- Qualifying Results (fastf1) ---[/bold cyan]\n")
    try:
        from pitwall.telemetry.session_data import get_qualifying_results
        with console.status(f"Loading qualifying for {year} R{round_num}..."):
            results = get_qualifying_results(year, round_num)

        console.print(f"[green]{len(results)} drivers:[/green]")
        for r in results:
            q1 = f"{r.q1_time:.3f}s" if r.q1_time else "-"
            q2 = f"{r.q2_time:.3f}s" if r.q2_time else "-"
            q3 = f"{r.q3_time:.3f}s" if r.q3_time else "-"
            console.print(f"  P{r.position:2d} {r.driver:4s} ({r.team:20s}) Q1: {q1:>10s}  Q2: {q2:>10s}  Q3: {q3:>10s}  [Out: {r.knocked_out_in}]")
        console.print("\n[bold green]Qualifying: OK[/bold green]")
    except Exception as e:
        console.print(f"[bold red]Qualifying: FAILED - {e}[/bold red]")


def test_telemetry_race(year: int, round_num: int):
    console.print("\n[bold cyan]--- Race Results (fastf1) ---[/bold cyan]\n")
    try:
        from pitwall.telemetry.session_data import get_race_results
        with console.status(f"Loading race for {year} R{round_num}..."):
            results = get_race_results(year, round_num)

        console.print(f"[green]{len(results)} drivers:[/green]")
        for r in results:
            fl = " [FL]" if r.fastest_lap else ""
            console.print(f"  P{r.position:2d} {r.driver:4s} ({r.team:20s}) Grid P{r.grid_position:2d} | {r.points:5.1f} pts | {r.status}{fl}")
        console.print("\n[bold green]Race Results: OK[/bold green]")
    except Exception as e:
        console.print(f"[bold red]Race Results: FAILED - {e}[/bold red]")


def test_telemetry_season(year: int, up_to_round: int):
    console.print("\n[bold cyan]--- Season Summary (fastf1) ---[/bold cyan]\n")
    try:
        from pitwall.telemetry.session_data import get_season_summary
        with console.status(f"Loading season {year} up to R{up_to_round}..."):
            summary = get_season_summary(year, up_to_round)

        console.print(f"[green]{summary.rounds_completed} rounds, {len(summary.driver_stats)} drivers:[/green]")
        for d in summary.driver_stats[:10]:
            console.print(f"  {d.driver:4s} ({d.team:20s}) {d.total_points:6.1f} pts, avg P{d.avg_finish_position:.1f}, {d.dnf_count} DNFs")
        console.print("\n[bold green]Season Summary: OK[/bold green]")
    except Exception as e:
        console.print(f"[bold red]Season Summary: FAILED - {e}[/bold red]")


def test_sprint(year: int, round_num: int):
    console.print("\n[bold cyan]--- Sprint Results (fastf1) ---[/bold cyan]\n")
    try:
        from pitwall.telemetry.session_data import get_sprint_results, get_sprint_qualifying_results
        with console.status(f"Loading sprint qualifying for {year} R{round_num}..."):
            sq_results = get_sprint_qualifying_results(year, round_num)

        console.print(f"[green]Sprint Qualifying — {len(sq_results)} drivers:[/green]")
        for r in sq_results:
            best = f"{r.q3_time:.3f}s" if r.q3_time else "-"
            console.print(f"  P{r.position:2d} {r.driver:4s} ({r.team:20s}) best: {best:>10s}  [Out: {r.knocked_out_in}]")

        with console.status(f"Loading sprint race for {year} R{round_num}..."):
            race_results = get_sprint_results(year, round_num)

        console.print(f"\n[green]Sprint Race — {len(race_results)} drivers:[/green]")
        for r in race_results:
            console.print(f"  P{r.position:2d} {r.driver:4s} ({r.team:20s}) Grid P{r.grid_position:2d} | {r.points:4.1f} pts | {r.status}")

        console.print("\n[bold green]Sprint: OK[/bold green]")
    except Exception as e:
        console.print(f"[bold red]Sprint: FAILED - {e}[/bold red]")


def test_historical(circuit: str):
    console.print("\n[bold cyan]--- Circuit History (fastf1) ---[/bold cyan]\n")
    try:
        from pitwall.telemetry.historical import get_circuit_history
        with console.status(f"Loading history for {circuit}..."):
            history = get_circuit_history(circuit, years=3)

        console.print(f"[green]{len(history)} years of data:[/green]")
        for h in history:
            top3 = ", ".join(r["driver"] for r in h["results"][:3])
            console.print(f"  {h['year']} {h['event_name']}: {top3}")
        console.print("\n[bold green]Circuit History: OK[/bold green]")
    except Exception as e:
        console.print(f"[bold red]Circuit History: FAILED - {e}[/bold red]")


def test_weather(city: str, country_code: str):
    console.print("\n[bold cyan]--- Weather Forecast ---[/bold cyan]\n")
    try:
        from pitwall.telemetry.weather import get_race_weekend_weather
        with console.status(f"Fetching weather for {city}, {country_code}..."):
            forecast = get_race_weekend_weather(city, country_code)

        if forecast.qualifying_day:
            q = forecast.qualifying_day
            console.print(f"  [green]Qualifying ({q.date}):[/green] {q.temp_celsius}C, {q.conditions}, rain {q.rain_probability:.0%}")
        else:
            console.print("  [yellow]Qualifying: no forecast available (too far out?)[/yellow]")

        if forecast.race_day:
            r = forecast.race_day
            console.print(f"  [green]Race ({r.date}):[/green] {r.temp_celsius}C, {r.conditions}, rain {r.rain_probability:.0%}")
        else:
            console.print("  [yellow]Race: no forecast available (too far out?)[/yellow]")

        console.print("\n[bold green]Weather: OK[/bold green]")
    except Exception as e:
        console.print(f"[bold red]Weather: FAILED - {e}[/bold red]")


def test_news():
    console.print("\n[bold cyan]--- F1 News (RSS) ---[/bold cyan]\n")
    try:
        from pitwall.telemetry.news import get_f1_news
        with console.status("Fetching RSS feeds..."):
            news = get_f1_news(days_back=5)

        console.print(f"[green]{len(news)} articles:[/green]")
        for n in news[:8]:
            console.print(f"  [{n.source}] {n.title}")
        if len(news) > 8:
            console.print(f"  ... and {len(news) - 8} more")
        console.print("\n[bold green]News: OK[/bold green]")
    except Exception as e:
        console.print(f"[bold red]News: FAILED - {e}[/bold red]")


def main():
    parser = argparse.ArgumentParser(description="Test pitwall modules without LLM")
    parser.add_argument("modules", nargs="*", default=["all"],
                        help="Modules to test: practice, quali, race, sprint, season, history, weather, news, all")
    parser.add_argument("--year", type=int, default=2024, help="Year for fastf1 data (default: 2024)")
    parser.add_argument("--round", type=int, default=1, help="Round for fastf1 data (default: 1)")
    parser.add_argument("--circuit", type=str, default="Melbourne", help="Circuit name for history")
    parser.add_argument("--city", type=str, default="Melbourne", help="City for weather")
    parser.add_argument("--country", type=str, default="AU", help="Country code for weather")
    parser.add_argument("--headed", action="store_true", help="Run browser in headed mode (visible window)")
    parser.add_argument("--slow-mo", type=int, default=0, help="Slow down browser actions by N ms (use with --headed)")
    args = parser.parse_args()

    console.print("[bold red]PITWALL[/bold red] [dim]- Module Test Runner[/dim]\n")

    targets = set(args.modules)
    run_all = "all" in targets

    # if run_all or "scraper" in targets:
    #     test_scraper(headless=not args.headed, slow_mo=args.slow_mo)
    if run_all or "practice" in targets:
        test_telemetry_practice(args.year, args.round)
    if run_all or "quali" in targets:
        test_telemetry_quali(args.year, args.round)
    if run_all or "race" in targets:
        test_telemetry_race(args.year, args.round)
    if "sprint" in targets:
        test_sprint(args.year, args.round)
    if run_all or "season" in targets:
        test_telemetry_season(args.year, args.round)
    if run_all or "history" in targets:
        test_historical(args.circuit)
    if run_all or "weather" in targets:
        test_weather(args.city, args.country)
    if run_all or "news" in targets:
        test_news()

    console.print("\n[bold]Done.[/bold]")


if __name__ == "__main__":
    main()

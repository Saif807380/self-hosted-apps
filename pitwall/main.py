import argparse
import os
import sys

from dotenv import load_dotenv
from rich.console import Console
from rich.markdown import Markdown

load_dotenv()

console = Console()


def parse_args():
    parser = argparse.ArgumentParser(description="Pitwall - F1 Fantasy Roster Optimizer")
    parser.add_argument("--year", type=int, help="Override RACE_YEAR from .env")
    parser.add_argument("--round", type=int, help="Override RACE_ROUND from .env")
    parser.add_argument("--dry-run", action="store_true", help="Use fixture data instead of live scraping/telemetry")
    parser.add_argument("--verbose", action="store_true", help="Print agent intermediate reasoning")
    return parser.parse_args()


def main():
    args = parse_args()

    # Apply overrides before importing config (which reads env at import time)
    if args.year:
        os.environ["RACE_YEAR"] = str(args.year)
    if args.round:
        os.environ["RACE_ROUND"] = str(args.round)

    console.print("[bold red]PITWALL[/bold red] [dim]- F1 Fantasy Roster Optimizer[/dim]\n")

    if args.dry_run:
        console.print("[yellow]Dry-run mode: using fixture data[/yellow]\n")

    # Validate config
    gemini_key = os.getenv("GEMINI_API_KEY")
    if not gemini_key:
        console.print("[red]Error: GEMINI_API_KEY is required. Set it in .env[/red]")
        sys.exit(1)

    # Select tools based on mode
    tools = None
    if args.dry_run:
        from agent.dry_run_tools import ALL_DRY_RUN_TOOLS
        tools = ALL_DRY_RUN_TOOLS

    from agent.orchestrator import run_agent

    with console.status("[bold cyan]Analyzing data and building roster recommendation...[/bold cyan]"):
        try:
            result = run_agent(verbose=args.verbose, tools=tools)
        except Exception as e:
            console.print(f"\n[red]Agent error: {e}[/red]")
            if args.verbose:
                import traceback
                console.print(traceback.format_exc())
            sys.exit(1)

    console.print()
    console.print(Markdown(result))


if __name__ == "__main__":
    main()

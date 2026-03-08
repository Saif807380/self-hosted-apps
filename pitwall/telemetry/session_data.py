from dataclasses import dataclass, field
import logging
import statistics

import fastf1
import pandas as pd

logging.getLogger("fastf1").setLevel(logging.ERROR)


@dataclass
class DriverPace:
    driver: str
    full_name: str
    team: str
    avg_long_run_pace: float
    lap_count: int
    consistency: float  # stdev of long-run lap times


@dataclass
class QualifyingResult:
    driver: str
    team: str
    position: int
    q1_time: float | None
    q2_time: float | None
    q3_time: float | None
    knocked_out_in: str  # "Q1", "Q2", "Q3" (reached final session)


@dataclass
class RaceResult:
    driver: str
    team: str
    position: int
    grid_position: int
    points: float
    status: str
    fastest_lap: bool


@dataclass
class DriverSeasonStats:
    driver: str
    team: str
    total_points: float
    avg_quali_position: float
    avg_finish_position: float
    dnf_count: int
    points_trend: list[float] = field(default_factory=list)  # last 3 races


@dataclass
class ConstructorSeasonStats:
    constructor: str
    total_points: float
    avg_finish_positions: list[float] = field(default_factory=list)
    reliability_dnfs: int = 0


@dataclass
class SeasonSummary:
    year: int
    rounds_completed: int
    driver_stats: list[DriverSeasonStats] = field(default_factory=list)
    constructor_stats: list[ConstructorSeasonStats] = field(default_factory=list)


MIN_LONG_RUN_LAPS = 5
SKIP_INITIAL_STINT_LAPS = 2


def _is_pit_lap(lap) -> bool:
    return not pd.isna(lap["PitInTime"]) or not pd.isna(lap["PitOutTime"])


def _get_long_run_stints(driver_laps) -> list[list[float]]:
    stints = []
    current_stint = []

    for _, lap in driver_laps.iterrows():
        # Pit out lap starts a new stint
        if not pd.isna(lap["PitOutTime"]):
            if len(current_stint) > SKIP_INITIAL_STINT_LAPS:
                stints.append(current_stint[SKIP_INITIAL_STINT_LAPS:])
            current_stint = []
            continue

        # Skip pit-in laps, inaccurate laps, and laps with no time
        if _is_pit_lap(lap):
            continue
        if pd.isna(lap["LapTime"]):
            continue
        if not lap.get("IsAccurate", True):
            continue

        current_stint.append(lap["LapTime"].total_seconds())

    # Final stint
    if len(current_stint) > SKIP_INITIAL_STINT_LAPS:
        stints.append(current_stint[SKIP_INITIAL_STINT_LAPS:])

    return [s for s in stints if len(s) >= MIN_LONG_RUN_LAPS]


def get_practice_pace(year: int, round_num: int) -> list[DriverPace]:
    results = []
    driver_long_run_times: dict[str, list[float]] = {}
    driver_best_lap: dict[str, float] = {}
    driver_all_times: dict[str, list[float]] = {}
    driver_info: dict[str, dict] = {}

    for session_name in ("FP1", "FP2"):
        try:
            session = fastf1.get_session(year, round_num, session_name)
            session.load()
        except Exception:
            continue

        for drv in session.drivers:
            laps = session.laps.pick_drivers(drv)
            if laps.empty:
                continue

            info = laps.iloc[0]
            driver_abbr = str(info["Driver"])
            driver_info.setdefault(driver_abbr, {
                "full_name": str(info.get("FullName", driver_abbr)),
                "team": str(info["Team"]),
            })

            # Collect all valid lap times for fallback
            for _, lap in laps.iterrows():
                if pd.isna(lap["LapTime"]):
                    continue
                if lap.get("IsAccurate", False):
                    secs = lap["LapTime"].total_seconds()
                    driver_all_times.setdefault(driver_abbr, []).append(secs)
                    prev = driver_best_lap.get(driver_abbr, float("inf"))
                    driver_best_lap[driver_abbr] = min(prev, secs)

            # Collect long-run stint times
            stints = _get_long_run_stints(laps)
            for stint in stints:
                driver_long_run_times.setdefault(driver_abbr, []).extend(stint)

    # Build results for all drivers
    for driver_abbr, info in driver_info.items():
        long_run = driver_long_run_times.get(driver_abbr, [])
        all_times = driver_all_times.get(driver_abbr, [])

        if long_run and len(long_run) >= MIN_LONG_RUN_LAPS:
            avg_pace = statistics.mean(long_run)
            lap_count = len(long_run)
            consistency = statistics.stdev(long_run) if len(long_run) > 1 else 0.0
        elif all_times:
            # Fallback: use all accurate laps
            avg_pace = statistics.mean(all_times)
            lap_count = len(all_times)
            consistency = statistics.stdev(all_times) if len(all_times) > 1 else 0.0
        else:
            continue

        results.append(DriverPace(
            driver=driver_abbr,
            full_name=info["full_name"],
            team=info["team"],
            avg_long_run_pace=round(avg_pace, 3),
            lap_count=lap_count,
            consistency=round(consistency, 3),
        ))

    results.sort(key=lambda d: d.avg_long_run_pace)
    return results


def _to_seconds(val) -> float | None:
    if val is None or pd.isna(val):
        return None
    try:
        return round(val.total_seconds(), 3)
    except Exception:
        return None


def get_qualifying_results(year: int, round_num: int) -> list[QualifyingResult]:
    session = fastf1.get_session(year, round_num, "Q")
    session.load(laps=False, telemetry=False, weather=False, messages=False)

    results = []
    for _, row in session.results.iterrows():
        q1 = _to_seconds(row.get("Q1"))
        q2 = _to_seconds(row.get("Q2"))
        q3 = _to_seconds(row.get("Q3"))

        if q3 is not None:
            knocked_out_in = "Q3"
        elif q2 is not None:
            knocked_out_in = "Q2"
        else:
            knocked_out_in = "Q1"

        results.append(QualifyingResult(
            driver=str(row["Abbreviation"]),
            team=str(row["TeamName"]),
            position=int(row["Position"]) if not pd.isna(row["Position"]) else 0,
            q1_time=q1,
            q2_time=q2,
            q3_time=q3,
            knocked_out_in=knocked_out_in,
        ))

    results.sort(key=lambda r: r.position if r.position > 0 else 99)
    return results


def get_race_results(year: int, round_num: int) -> list[RaceResult]:
    session = fastf1.get_session(year, round_num, "R")
    session.load(laps=False, telemetry=False, weather=False, messages=False)

    results = []
    for _, row in session.results.iterrows():
        results.append(RaceResult(
            driver=str(row["Abbreviation"]),
            team=str(row["TeamName"]),
            position=int(row["Position"]) if str(row["Position"]) != "nan" else 0,
            grid_position=int(row["GridPosition"]) if str(row["GridPosition"]) != "nan" else 0,
            points=float(row["Points"]) if not pd.isna(row["Points"]) else 0.0,
            status=str(row.get("Status", "")),
            fastest_lap=bool(row.get("FastestLap", False)),
        ))

    results.sort(key=lambda r: r.position if r.position > 0 else 99)
    return results


def get_season_summary(year: int, up_to_round: int) -> SeasonSummary:
    driver_points: dict[str, float] = {}
    driver_teams: dict[str, str] = {}
    driver_quali_pos: dict[str, list[int]] = {}
    driver_finish_pos: dict[str, list[int]] = {}
    driver_dnfs: dict[str, int] = {}
    driver_recent_points: dict[str, list[float]] = {}

    constructor_points: dict[str, float] = {}
    constructor_finishes: dict[str, list[int]] = {}
    constructor_dnfs: dict[str, int] = {}

    rounds_completed = 0

    for rnd in range(1, up_to_round+1):
        # Race results
        try:
            race_results = get_race_results(year, rnd)
        except Exception:
            continue

        # Skip rounds where the race hasn't happened yet (fastf1 returns the
        # session from the schedule but all positions/points are zero)
        if not race_results or all(r.position == 0 for r in race_results):
            continue

        rounds_completed += 1

        for r in race_results:
            driver_points[r.driver] = driver_points.get(r.driver, 0) + r.points
            driver_teams[r.driver] = r.team
            driver_finish_pos.setdefault(r.driver, []).append(r.position)
            driver_recent_points.setdefault(r.driver, []).append(r.points)

            constructor_points[r.team] = constructor_points.get(r.team, 0) + r.points
            if r.position > 0:
                constructor_finishes.setdefault(r.team, []).append(r.position)

            is_dnf = r.status and r.status not in ("Finished", "+1 Lap", "+2 Laps", "+3 Laps")
            if is_dnf:
                driver_dnfs[r.driver] = driver_dnfs.get(r.driver, 0) + 1
                constructor_dnfs[r.team] = constructor_dnfs.get(r.team, 0) + 1

        # Qualifying results
        try:
            quali_results = get_qualifying_results(year, rnd)
            for q in quali_results:
                driver_quali_pos.setdefault(q.driver, []).append(q.position)
        except Exception:
            pass

    driver_stats = []
    for drv, pts in driver_points.items():
        quali_positions = driver_quali_pos.get(drv, [])
        finish_positions = driver_finish_pos.get(drv, [])
        recent = driver_recent_points.get(drv, [])

        driver_stats.append(DriverSeasonStats(
            driver=drv,
            team=driver_teams.get(drv, ""),
            total_points=pts,
            avg_quali_position=statistics.mean(quali_positions) if quali_positions else 0.0,
            avg_finish_position=statistics.mean(finish_positions) if finish_positions else 0.0,
            dnf_count=driver_dnfs.get(drv, 0),
            points_trend=recent[-3:],
        ))

    driver_stats.sort(key=lambda d: d.total_points, reverse=True)

    constructor_stats = []
    for team, pts in constructor_points.items():
        finishes = constructor_finishes.get(team, [])
        constructor_stats.append(ConstructorSeasonStats(
            constructor=team,
            total_points=pts,
            avg_finish_positions=[statistics.mean(finishes)] if finishes else [],
            reliability_dnfs=constructor_dnfs.get(team, 0),
        ))

    constructor_stats.sort(key=lambda c: c.total_points, reverse=True)

    return SeasonSummary(
        year=year,
        rounds_completed=rounds_completed,
        driver_stats=driver_stats,
        constructor_stats=constructor_stats,
    )

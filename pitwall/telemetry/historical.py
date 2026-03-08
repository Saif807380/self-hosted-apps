import logging

import fastf1

logging.getLogger("fastf1").setLevel(logging.ERROR)


def get_circuit_history(circuit_name: str, years: int = 3) -> list[dict]:
    import config.config as config
    current_year = config.RACE_YEAR
    results = []

    for year in range(current_year - years, current_year):
        try:
            schedule = fastf1.get_event_schedule(year)
            event = schedule[
                schedule["EventName"].str.contains(circuit_name, case=False, na=False)
                | schedule["Location"].str.contains(circuit_name, case=False, na=False)
                | schedule["Country"].str.contains(circuit_name, case=False, na=False)
            ]
            if event.empty:
                continue

            round_num = int(event.iloc[0]["RoundNumber"])
            session = fastf1.get_session(year, round_num, "R")
            session.load(laps=False, telemetry=False, weather=False, messages=False)

            race_results = []
            for _, row in session.results.iterrows():
                race_results.append({
                    "driver": str(row["Abbreviation"]),
                    "team": str(row["TeamName"]),
                    "position": int(row["Position"]) if str(row["Position"]) != "nan" else 0,
                    "points": float(row["Points"]),
                })

            results.append({
                "year": year,
                "circuit": circuit_name,
                "event_name": str(event.iloc[0]["EventName"]),
                "results": race_results,
            })
        except Exception:
            continue

    return results

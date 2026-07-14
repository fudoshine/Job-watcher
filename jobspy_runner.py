#!/usr/bin/env python3
import argparse
import json
import logging
import os
import sys
from datetime import date, datetime

import pandas as pd


def add_local_jobspy():
    repo_root = os.path.dirname(os.path.abspath(__file__))
    local_jobspy = os.path.join(repo_root, "JobSpy")
    if os.path.isdir(local_jobspy) and local_jobspy not in sys.path:
        sys.path.insert(0, local_jobspy)


def parse_csv(value):
    if not value:
        return None
    return [item.strip() for item in value.split(",") if item.strip()]


def parse_bool(value):
    if value is None:
        return None
    if isinstance(value, bool):
        return value
    value = value.strip().lower()
    if value in ("1", "true", "yes", "y", "on"):
        return True
    if value in ("0", "false", "no", "n", "off"):
        return False
    raise argparse.ArgumentTypeError("Expected a boolean value (true/false).")


def normalize_value(value):
    if value is None:
        return None
    try:
        if pd.isna(value):
            return None
    except Exception:
        pass
    if isinstance(value, (pd.Timestamp, datetime, date)):
        return value.isoformat()
    return value


def main():
    parser = argparse.ArgumentParser(description="Run JobSpy and emit JSON results.")
    parser.add_argument("--sites", type=parse_csv)
    parser.add_argument("--search-term")
    parser.add_argument("--google-search-term")
    parser.add_argument("--location")
    parser.add_argument("--distance", type=int)
    parser.add_argument("--is-remote", action="store_true")
    parser.add_argument("--job-type")
    parser.add_argument("--easy-apply", type=parse_bool)
    parser.add_argument("--results-wanted", type=int)
    parser.add_argument("--country-indeed")
    parser.add_argument("--hours-old", type=int)
    parser.add_argument("--linkedin-fetch-description", action="store_true")
    parser.add_argument("--offset", type=int)
    parser.add_argument("--user-agent")
    parser.add_argument("--proxies", type=parse_csv)
    parser.add_argument("--verbose", type=int, default=0)
    args = parser.parse_args()

    logging.basicConfig(stream=sys.stderr, level=logging.ERROR)

    add_local_jobspy()
    try:
        from jobspy import scrape_jobs
    except Exception as exc:
        print(f"JobSpy import failed: {exc}", file=sys.stderr)
        sys.exit(1)

    jobs_df = scrape_jobs(
        site_name=args.sites,
        search_term=args.search_term,
        google_search_term=args.google_search_term,
        location=args.location,
        # Passing None overrides JobSpy's own default and generates an invalid
        # Indeed GraphQL value ("radius: None").
        distance=args.distance if args.distance is not None else 50,
        is_remote=args.is_remote,
        job_type=args.job_type,
        easy_apply=args.easy_apply,
        results_wanted=args.results_wanted or 15,
        country_indeed=args.country_indeed or "usa",
        proxies=args.proxies,
        linkedin_fetch_description=args.linkedin_fetch_description,
        offset=args.offset or 0,
        hours_old=args.hours_old,
        user_agent=args.user_agent,
        verbose=args.verbose,
    )

    if jobs_df is None or jobs_df.empty:
        print("[]")
        return

    records = []
    for record in jobs_df.to_dict(orient="records"):
        normalized = {key: normalize_value(value) for key, value in record.items()}
        records.append(normalized)

    json.dump(records, sys.stdout, ensure_ascii=False)


if __name__ == "__main__":
    main()

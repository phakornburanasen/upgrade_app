import argparse
import json
import re
import sys
import time
import urllib.error
import urllib.request


def post_json(url, payload, timeout):
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    return request_json(req, timeout)


def get_json(url, timeout):
    return request_json(urllib.request.Request(url, method="GET"), timeout)


def request_json(req, timeout):
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            body = resp.read().decode("utf-8")
            return json.loads(body)
    except urllib.error.HTTPError as err:
        body = err.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {err.code}: {body}") from err
    except urllib.error.URLError as err:
        raise RuntimeError(f"Connection failed: {err.reason}") from err


def normalize_agent_url(value):
    value = value.strip()
    match = re.fullmatch(r"\[[^\]]+\]\((https?://[^)]+)\)", value)
    if match:
        value = match.group(1)
    if not value.startswith(("http://", "https://")):
        value = "http://" + value
    return value.rstrip("/")


def main():
    parser = argparse.ArgumentParser(description="Send update request to local UpdateAgent.")
    parser.add_argument(
        "applications",
        nargs="+",
        help="Application/folder names to update, for example: car Agent_TNLX",
    )
    parser.add_argument(
        "--agent",
        default="http://127.0.0.1:45100",
        help="UpdateAgent URL. Default: http://127.0.0.1:45100",
    )
    parser.add_argument(
        "--no-wait",
        action="store_true",
        help="Start the update and exit without polling progress.",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=10,
        help="HTTP timeout in seconds. Default: 10",
    )
    args = parser.parse_args()

    agent_url = normalize_agent_url(args.agent)
    apps = [app.strip() for app in args.applications if app.strip()]
    if not apps:
        print("ERROR: Please provide at least one application name.", file=sys.stderr)
        return 1

    print(f"Sending update request to {agent_url}")
    print("Applications:", ", ".join(apps))

    try:
        started = post_json(f"{agent_url}/api/update", {"applications": apps}, args.timeout)
    except RuntimeError as err:
        print(f"ERROR: {err}", file=sys.stderr)
        print("Check that update-agent.exe is running and the agent host/port is reachable.", file=sys.stderr)
        return 1
    job_id = started.get("data", {}).get("job_id")
    if not job_id:
        print(json.dumps(started, indent=2))
        print("ERROR: Response did not include job_id.", file=sys.stderr)
        return 1

    print(f"Job started: {job_id}")
    if args.no_wait:
        return 0

    while True:
        try:
            status = get_json(f"{agent_url}/api/update/status/{job_id}", args.timeout)
        except RuntimeError as err:
            print(f"ERROR: {err}", file=sys.stderr)
            return 1
        job = status.get("data", {})
        total_bytes = job.get("total_bytes") or 0
        completed_bytes = job.get("completed_bytes") or 0
        percent = int((completed_bytes / total_bytes) * 100) if total_bytes else 0
        current_app = job.get("current_app") or "-"
        message = job.get("message") or ""
        state = job.get("status") or "unknown"
        completed_files = job.get("completed_files") or 0
        total_files = job.get("total_files") or 0

        print(
            f"[{state}] {current_app} | {message} | "
            f"{completed_files}/{total_files} files | {percent}%"
        )

        if state in ("success", "failed"):
            results = job.get("app_results") or []
            if results:
                print("\nSummary:")
                for result in results:
                    app = result.get("application", "-")
                    status_text = result.get("status", "-")
                    error = result.get("error")
                    if error:
                        print(f"  {app}: {status_text} - {error}")
                    else:
                        print(f"  {app}: {status_text}")
            return 0 if state == "success" else 2

        time.sleep(1)


if __name__ == "__main__":
    raise SystemExit(main())

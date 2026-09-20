import argparse
import json
import sys
import urllib.error
import urllib.request


def get_json(url, timeout):
    try:
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            return resp.status, json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as err:
        body = err.read().decode("utf-8", errors="replace")
        return err.code, body
    except urllib.error.URLError as err:
        return None, f"Connection failed: {err.reason}"


def print_check(label, url, timeout):
    print(f"\n== {label} ==")
    print(url)
    status, data = get_json(url, timeout)
    if status is None:
        print(data)
        return False
    print(f"HTTP {status}")
    if isinstance(data, str):
        print(data)
    else:
        print(json.dumps(data, indent=2))
    return 200 <= status < 300


def main():
    parser = argparse.ArgumentParser(description="Diagnose Update Server and UpdateAgent connectivity.")
    parser.add_argument("--server", default="http://127.0.0.1:45000", help="Update server URL")
    parser.add_argument("--agent", default="http://127.0.0.1:45100", help="Update agent URL")
    parser.add_argument("--app", default="car", help="Application/folder name to check")
    parser.add_argument("--timeout", type=int, default=5, help="HTTP timeout in seconds")
    args = parser.parse_args()

    server = args.server.rstrip("/")
    agent = args.agent.rstrip("/")
    app = args.app.strip()

    ok = True
    ok = print_check("UpdateAgent local status", f"{agent}/api/local/status", args.timeout) and ok
    ok = print_check("UpdateAgent apps via server", f"{agent}/api/apps", args.timeout) and ok
    ok = print_check("Update Server health", f"{server}/api/health", args.timeout) and ok
    ok = print_check("Update Server apps", f"{server}/api/apps", args.timeout) and ok
    ok = print_check("Update Server file list for app", f"{server}/api/apps/{app}/files", args.timeout) and ok

    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main())

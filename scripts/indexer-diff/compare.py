#!/usr/bin/env python3
"""Run synthetic scenarios through Moji and the pinned actual Jackett engine.

No production indexer or user configuration is loaded. The reference process
runs without networking; Moji has only a fixture RoundTripper. Raw results and
explicitly classified differences are retained in the private report.
"""

import argparse
import base64
import copy
from datetime import datetime
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True
from build_reference import DIRECTORY, run_container, verify_provenance

ROOT = DIRECTORY.parents[1]
MISSING = object()


def case_file(root, relative, limit):
    path = (root / relative).resolve()
    if not path.is_relative_to(root.resolve()) or path.stat().st_size > limit:
        raise ValueError("fixture file outside case or exceeds budget")
    return path.read_bytes()


def resolve_case(path):
    if path.stat().st_size > 1 << 20:
        raise ValueError("scenario exceeds metadata budget")
    data = json.loads(path.read_text())
    if "relativeDateFilters" in data:
        if set(data) != {"relativeDateFilters", "now"} or not isinstance(data["now"], str):
            raise ValueError("relative-date helper requires only a fixed clock and inputs")
        clock = datetime.fromisoformat(data["now"].replace("Z", "+00:00"))
        rows = data["relativeDateFilters"]
        if clock.tzinfo is None or not isinstance(rows, list) or not 1 <= len(rows) <= 100:
            raise ValueError("relative-date helper requires a zoned clock and 1..100 inputs")
        for row in rows:
            if not isinstance(row, dict) or set(row) - {"name", "input", "expectInvalid"} or row.get("name") not in {"timeago", "reltime"} or not isinstance(row.get("input"), str) or len(row["input"].encode("utf-8")) > 4096 or not isinstance(row.get("expectInvalid", False), bool):
                raise ValueError("invalid relative-date helper input")
        return data, [], None
    settings = data.get("settings", {})
    if not isinstance(settings, dict) or any(not isinstance(value, str) for value in settings.values()):
        raise ValueError("settings must be a string-to-string map at the interpreter boundary")
    allowed = {"definitionFile", "query", "settings", "authenticate", "challengeAnswer", "fetch", "now", "responses", "expectedDifferences", "expectedErrors"}
    if set(data) - allowed:
        raise ValueError("unknown scenario metadata")
    if "challengeAnswer" in data and (data.get("authenticate") is not True or not isinstance(data["challengeAnswer"], str) or not data["challengeAnswer"].strip() or len(data["challengeAnswer"].encode("utf-8")) > 4096):
        raise ValueError("challengeAnswer requires authentication and a bounded nonempty string")
    expected = data.pop("expectedDifferences", [])
    expected_errors = data.pop("expectedErrors", None)
    if expected_errors is not None and (set(expected_errors) != {"moji", "jackett"} or not all(isinstance(v, str) and v for v in expected_errors.values())):
        raise ValueError("expectedErrors requires exact nonempty errors for both engines")
    data["definition"] = case_file(path.parent, data.pop("definitionFile"), 1 << 20).decode("utf-8")
    if len(data["responses"]) > 100:
        raise ValueError("response count exceeds budget")
    total = 0
    for response in data["responses"]:
        if ("bodyFile" in response) == ("bodyText" in response):
            raise ValueError("choose one response body source")
        body = case_file(path.parent, response.pop("bodyFile"), 16 << 20) if "bodyFile" in response else response.pop("bodyText").encode("utf-8")
        total += len(body)
        if total > 20 << 20:
            raise ValueError("response bytes exceed scenario budget")
        response["body"] = base64.b64encode(body).decode("ascii")
    return data, expected, expected_errors


def normalized(report):
    value = copy.deepcopy(report)
    value.pop("engine", None)
    defaults = []
    for i, request in enumerate(value.get("requests", [])):
        headers = {name.lower(): item for name, item in request.get("headers", {}).items()}
        for name, default in [("accept", "*/*"), ("accept-language", "*")]:
            if headers.get(name) == default:
                defaults.append({"request": i, "header": name, "value": headers.pop(name)})
        request["headers"] = headers
    for result in value.get("results", []):
        timestamp = result.get("publishedAt")
        if timestamp:
            timestamp = timestamp.replace("+00:00", "Z")
            result["publishedAt"] = re.sub(r"\.([0-9]+)Z$", lambda m: ("." + m[1].rstrip("0") if m[1].rstrip("0") else "") + "Z", timestamp)
    return value, defaults


def differences(left, right, path=""):
    if left is MISSING or right is MISSING:
        return [{"path": path, "moji": {"missing": True} if left is MISSING else left,
                 "jackett": {"missing": True} if right is MISSING else right}]
    if isinstance(left, dict) and isinstance(right, dict):
        result = []
        for key in sorted(set(left) | set(right)):
            pointer = str(key).replace("~", "~0").replace("/", "~1")
            result.extend(differences(left.get(key, MISSING), right.get(key, MISSING), path + "/" + pointer))
        return result
    if isinstance(left, list) and isinstance(right, list):
        if len(left) != len(right):
            return [{"path": path, "moji": left, "jackett": right}]
        return [item for i, pair in enumerate(zip(left, right)) for item in differences(*pair, path + "/" + str(i))]
    return [] if left == right else [{"path": path, "moji": left, "jackett": right}]


def classify(moji, jackett, expected, expected_errors=None):
    left, moji_defaults = normalized(moji)
    right, jackett_defaults = normalized(jackett)
    errors_match = expected_errors is not None and all(report.get("error") == expected_errors[engine] for engine, report in [("moji", moji), ("jackett", jackett)])
    if errors_match:
        left.pop("error", None)
        right.pop("error", None)
        left.setdefault("results", [])
        right.setdefault("results", [])
    actual = differences(left, right)
    unexpected, accepted = [], []
    used = set()
    for item in actual:
        match = next((i for i, entry in enumerate(expected) if entry.get("reason") and
                      all(entry.get(key, MISSING) == item[key] for key in ["path", "moji", "jackett"])), None)
        if match is None:
            unexpected.append(item)
        else:
            used.add(match)
            accepted.append({**item, "reason": expected[match]["reason"]})
    stale = [item for i, item in enumerate(expected) if i not in used]
    return {"unexpected": unexpected, "accepted": accepted, "staleExpectations": stale,
            "expectedErrors": expected_errors, "errorsMatched": errors_match,
            "transportDefaults": {"moji": moji_defaults, "jackett": jackett_defaults}}


def case_passed(moji, jackett, comparison):
    execution_failed = bool(moji.get("error") or jackett.get("error"))
    outcome_matches = comparison["errorsMatched"] if comparison["expectedErrors"] is not None else not execution_failed
    return outcome_matches and not (comparison["unexpected"] or comparison["staleExpectations"])


def run_cases(reference_root, cases, output):
    provenance = verify_provenance(reference_root)
    reports = []
    with tempfile.TemporaryDirectory(prefix="moji-run-", dir=reference_root) as replay_directory, \
         tempfile.TemporaryDirectory(prefix="moji-indexer-replay-") as temporary:
        binary = Path(temporary) / "moji-indexer-replay"
        subprocess.run(["go", "build", "-o", str(binary), "./cmd/moji-indexer-replay"], cwd=ROOT, check=True)
        for case in cases:
            resolved, expected, expected_errors = resolve_case(case)
            # The reference root contains only downloaded source, generated
            # binaries and these synthetic scenario/results, never user data.
            # Each comparison owns its files, even if another process uses the
            # same immutable reference binaries concurrently.
            replay = Path(replay_directory)
            scenario_path = replay / "scenario.json"
            moji_path, jackett_path = replay / "moji-result.json", replay / "jackett-result.json"
            scenario_path.write_text(json.dumps(resolved, ensure_ascii=True))
            subprocess.run([str(binary), "-scenario", str(scenario_path), "-output", str(moji_path)], check=True, timeout=40)
            command = ["dotnet", "moji-reference/bin/Release/net9.0/IndexerReference.dll",
                       scenario_path.relative_to(reference_root).as_posix(), jackett_path.relative_to(reference_root).as_posix()]
            run_container(reference_root, command, "none", 60)
            moji, jackett = json.loads(moji_path.read_text()), json.loads(jackett_path.read_text())
            comparison = classify(moji, jackett, expected, expected_errors)
            passed = case_passed(moji, jackett, comparison)
            reports.append({"case": case.parent.name, "passed": passed, "moji": moji, "jackett": jackett, **comparison})
            print(f"{case.parent.name}: {'PASS' if passed else 'FAIL'}; "
                  f"{len(comparison['unexpected'])} unexpected, {len(comparison['accepted'])} explained differences")
    result = {"schemaVersion": 1, "reference": provenance, "cases": reports}
    output.write_text(json.dumps(result, indent=2, ensure_ascii=True) + "\n")
    output.chmod(0o600)
    return all(item["passed"] for item in reports)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--reference-root", required=True, type=Path)
    parser.add_argument("--case", action="append", type=Path)
    parser.add_argument("--report", required=True, type=Path)
    args = parser.parse_args()
    cases = args.case or sorted((DIRECTORY / "cases").glob("*/scenario.json"))
    if not cases:
        parser.error("no scenarios selected")
    passed = run_cases(args.reference_root.resolve(), cases, args.report.resolve())
    raise SystemExit(0 if passed else 1)


if __name__ == "__main__":
    main()

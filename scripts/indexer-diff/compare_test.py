"""Check that the differential checker cannot silently accept a changed result."""

import copy
from concurrent.futures import ThreadPoolExecutor
import json
from pathlib import Path
import sys
import tempfile
import threading
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
import compare


class ComparisonTests(unittest.TestCase):
    def test_relative_date_helper_requires_a_fixed_clock_and_isolated_bounded_inputs(self):
        with tempfile.TemporaryDirectory() as directory:
            scenario = Path(directory) / "scenario.json"
            valid = {"now": "2024-03-31T12:34:56.5Z", "relativeDateFilters": [{"name": "timeago", "input": "1 month"}]}
            scenario.write_text(json.dumps(valid))
            self.assertEqual(compare.resolve_case(scenario), (valid, [], None))
            fuzzy = {**valid, "relativeDateFilters": [{"name": "fuzzytime", "input": "today"}]}
            scenario.write_text(json.dumps(fuzzy))
            self.assertEqual(compare.resolve_case(scenario), (fuzzy, [], None))
            for invalid in [{**valid, "now": "2024-03-31T12:34:56"},
                            {**valid, "responses": []},
                            {**valid, "relativeDateFilters": []},
                            {**valid, "relativeDateFilters": valid["relativeDateFilters"] * 101},
                            {**valid, "relativeDateFilters": [{"name": "execute", "input": "1 month"}]},
                            {**valid, "relativeDateFilters": [{"name": "timeago", "input": "x" * 4097}]},
                            {**valid, "relativeDateFilters": [{"name": "timeago", "input": "1 month", "expectInvalid": "yes"}]}]:
                scenario.write_text(json.dumps(invalid))
                with self.assertRaises(ValueError):
                    compare.resolve_case(scenario)

    def test_relative_date_values_and_failure_status_are_compared_exactly(self):
        left = {"engine": "synthetic", "requests": [], "results": [], "mode": "relative-date-helper",
                "relativeDates": [{"name": "timeago", "input": "1 month", "value": "Fri, 01 Mar 2024 12:34:56 +0", "invalid": False}]}
        right = copy.deepcopy(left)
        right["relativeDates"][0]["value"] = "Thu, 29 Feb 2024 12:34:56 +0"
        self.assertEqual(compare.classify(left, right, [])["unexpected"][0]["path"], "/relativeDates/0/value")
        right = copy.deepcopy(left)
        right["relativeDates"][0]["invalid"] = True
        self.assertEqual(compare.classify(left, right, [])["unexpected"][0]["path"], "/relativeDates/0/invalid")

    def test_concurrent_runs_cannot_share_scenario_or_result_files(self):
        barrier = threading.Barrier(2)
        scenario_paths = set()
        lock = threading.Lock()

        def report(marker):
            return {"engine": "synthetic", "requests": [], "results": [{"title": marker}]}

        def process(command, **kwargs):
            if command[0] != "go":
                scenario = Path(command[command.index("-scenario") + 1])
                output = Path(command[command.index("-output") + 1])
                output.write_text(json.dumps(report(json.loads(scenario.read_text())["marker"])))

        def reference(root, command, network, timeout):
            scenario, output = root / command[2], root / command[3]
            with lock:
                scenario_paths.add(scenario)
            barrier.wait(timeout=5)
            output.write_text(json.dumps(report(json.loads(scenario.read_text())["marker"])))

        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with patch.object(compare, "verify_provenance", return_value={"source": "synthetic"}), \
                 patch.object(compare, "resolve_case", side_effect=lambda case: ({"marker": case.parent.name}, [], None)), \
                 patch.object(compare.subprocess, "run", side_effect=process), \
                 patch.object(compare, "run_container", side_effect=reference):
                with ThreadPoolExecutor(max_workers=2) as workers:
                    runs = [workers.submit(compare.run_cases, root, [root / marker / "scenario.json"], root / (marker + ".json")) for marker in ["left", "right"]]
                    self.assertTrue(all(run.result(timeout=10) for run in runs))
            self.assertEqual(len(scenario_paths), 2)
            self.assertTrue(all(not path.exists() for path in scenario_paths))
            for marker in ["left", "right"]:
                case = json.loads((root / (marker + ".json")).read_text())["cases"][0]
                self.assertEqual(case["moji"]["results"], [{"title": marker}])
                self.assertEqual(case["jackett"]["results"], [{"title": marker}])

    def report(self):
        return {"engine": "fixture", "requests": [{"method": "GET", "url": "https://fixture.invalid/", "body": "", "headers": {}}],
                "results": [{"title": "ABCD-123", "publishedAt": None}]}

    def test_transport_defaults_are_explained_and_raw_input_is_preserved(self):
        left, right = self.report(), self.report()
        right["requests"][0]["headers"] = {"Accept": "*/*", "Accept-Language": "*"}
        before = copy.deepcopy(right)
        result = compare.classify(left, right, [])
        self.assertEqual(result["unexpected"], [])
        self.assertEqual(len(result["transportDefaults"]["jackett"]), 2)
        self.assertEqual(right, before)

    def test_custom_accept_header_is_not_hidden(self):
        left, right = self.report(), self.report()
        right["requests"][0]["headers"] = {"Accept": "application/json"}
        result = compare.classify(left, right, [])
        self.assertEqual(result["unexpected"][0]["path"], "/requests/0/headers/accept")

    def test_request_body_bytes_are_compared(self):
        left, right = self.report(), self.report()
        right["requests"][0]["body"] = "dGVzdA=="
        result = compare.classify(left, right, [])
        self.assertEqual(result["unexpected"][0]["path"], "/requests/0/body")

    def test_challenge_presentation_is_compared(self):
        left, right = self.report(), self.report()
        left["challenge"] = {"type": "image", "prompt": "", "imageSha256": "first"}
        right["challenge"] = {"type": "image", "prompt": "", "imageSha256": "second"}
        result = compare.classify(left, right, [])
        self.assertEqual(result["unexpected"][0]["path"], "/challenge/imageSha256")

    def test_missing_artifact_is_one_explicit_difference(self):
        left, right = self.report(), self.report()
        right["artifact"] = {"sha256": "synthetic", "magnet": ""}
        result = compare.classify(left, right, [])
        self.assertEqual(result["unexpected"], [{"path": "/artifact", "moji": {"missing": True}, "jackett": right["artifact"]}])

    def test_magnet_tracker_parameters_are_compared_exactly(self):
        left, right = self.report(), self.report()
        left["artifact"] = {"sha256": "", "magnet": "magnet:?xt=urn:btih:synthetic&tr=one"}
        right["artifact"] = {"sha256": "", "magnet": "magnet:?xt=urn:btih:synthetic&tr=two"}
        result = compare.classify(left, right, [])
        self.assertEqual(result["unexpected"][0]["path"], "/artifact/magnet")

    def test_encoded_and_display_unicode_magnets_are_not_normalized(self):
        left, right = self.report(), self.report()
        left["artifact"] = {"sha256": "", "magnet": "magnet:?xt=urn:btih:synthetic&dn=%E6%B5%8B%E8%AF%95"}
        right["artifact"] = {"sha256": "", "magnet": "magnet:?xt=urn:btih:synthetic&dn=测试"}
        self.assertEqual(compare.classify(left, right, [])["unexpected"][0]["path"], "/artifact/magnet")

    def test_same_timestamp_with_different_fraction_format_is_equal(self):
        left, right = self.report(), self.report()
        left["results"][0]["publishedAt"] = "2026-11-03T00:00:00Z"
        right["results"][0]["publishedAt"] = "2026-11-03T00:00:00.0000000+00:00"
        self.assertEqual(compare.classify(left, right, [])["unexpected"], [])

    def test_named_difference_requires_exact_values_and_reason(self):
        left, right = self.report(), self.report()
        right["results"][0]["title"] = "Other"
        expected = [{"path": "/results/0/title", "moji": "ABCD-123", "jackett": "Other", "reason": "explicit fixture contract"}]
        self.assertEqual(len(compare.classify(left, right, expected)["accepted"]), 1)
        expected[0]["jackett"] = "Different"
        result = compare.classify(left, right, expected)
        self.assertEqual(len(result["unexpected"]), 1)
        self.assertEqual(len(result["staleExpectations"]), 1)

    def test_removed_difference_invalidates_old_expectation(self):
        report = self.report()
        expected = [{"path": "/results/0/title", "moji": "ABCD-123", "jackett": "Other", "reason": "old contract"}]
        self.assertEqual(len(compare.classify(report, report, expected)["staleExpectations"]), 1)

    def test_engine_errors_require_exact_declared_outcomes(self):
        left, right = self.report(), self.report()
        left["error"], right["error"] = "invalid episode", "OverflowException"
        comparison = compare.classify(left, right, [])
        self.assertFalse(compare.case_passed(left, right, comparison))
        expected = {"moji": "invalid episode", "jackett": "OverflowException"}
        comparison = compare.classify(left, right, [], expected)
        self.assertTrue(compare.case_passed(left, right, comparison))
        right["requests"].append(copy.deepcopy(right["requests"][0]))
        comparison = compare.classify(left, right, [], expected)
        self.assertFalse(compare.case_passed(left, right, comparison))

    def test_stale_expected_error_does_not_pass_success(self):
        report = self.report()
        expected = {"moji": "old error", "jackett": "old error"}
        comparison = compare.classify(report, report, [], expected)
        self.assertFalse(compare.case_passed(report, report, comparison))

    def test_explained_request_change_does_not_hide_artifact_change(self):
        left, right = self.report(), self.report()
        left["artifact"], right["artifact"] = {"sha256": "first"}, {"sha256": "second"}
        right["requests"].append(copy.deepcopy(right["requests"][0]))
        expected = [{"path": "/requests", "moji": left["requests"], "jackett": right["requests"], "reason": "no duplicate acquisition"}]
        result = compare.classify(left, right, expected)
        self.assertEqual(len(result["accepted"]), 1)
        self.assertEqual([item["path"] for item in result["unexpected"]], ["/artifact/sha256"])

    def test_fixture_file_cannot_escape_case_directory(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            case = root / "case"
            case.mkdir()
            (root / "outside.yml").write_text("synthetic")
            path = case / "scenario.json"
            path.write_text(json.dumps({"definitionFile": "../outside.yml", "responses": []}))
            with self.assertRaisesRegex(ValueError, "outside case"):
                compare.resolve_case(path)

    def test_challenge_answer_requires_authentication_and_bounded_text(self):
        source = compare.DIRECTORY / "cases/captcha-text/scenario.json"
        original = json.loads(source.read_text())
        for answer, authenticate in [("7", False), (None, True), (True, True), (" ", True), ("界" * 1366, True)]:
            with self.subTest(answer_type=type(answer).__name__, authenticate=authenticate), tempfile.TemporaryDirectory() as temporary:
                path = Path(temporary) / "scenario.json"
                data = copy.deepcopy(original)
                data["challengeAnswer"], data["authenticate"] = answer, authenticate
                path.write_text(json.dumps(data))
                with self.assertRaisesRegex(ValueError, "challengeAnswer"):
                    compare.resolve_case(path)

    def test_settings_match_interpreter_string_boundary(self):
        source = json.loads((compare.DIRECTORY / "cases/public-json/scenario.json").read_text())
        for settings in [[], {"enabled": True}, {"enabled": 1}, {"enabled": None}]:
            with self.subTest(settings=settings), tempfile.TemporaryDirectory() as temporary:
                path = Path(temporary) / "scenario.json"
                data = copy.deepcopy(source)
                data["settings"] = settings
                path.write_text(json.dumps(data))
                with self.assertRaisesRegex(ValueError, "settings must be a string-to-string map"):
                    compare.resolve_case(path)

    def test_all_shipped_scenarios_resolve(self):
        cases = list((compare.DIRECTORY / "cases").glob("*/scenario.json"))
        required = {"public-html", "public-json", "post-login", "download-page", "download-before", "tested-download",
                    "tv-episode", "tv-season", "tv-daily", "tv-episode-suffix", "tv-no-season",
                    "tv-episode-fraction", "tv-episode-grouped", "tv-episode-text", "tv-daily-invalid",
                    "tv-episode-nonintegral-error", "tv-leading-separator", "tv-separator-in-fraction",
                    "movie-query", "music-query", "book-query", "captcha-text", "captcha-image", "captcha-text-rejected",
                    "query-keys-get", "query-keys-post", "query-pagination-unset", "query-pagination-zero", "query-pagination-values",
                    "query-ids", "query-ids-empty", "query-id-overflow-error",
                    "magnet-download", "magnet-field", "magnet-redirect", "magnet-page", "magnet-before", "query-cjk-keywords",
                    "simple-captcha", "andmatch-basic", "andmatch-limit", "andmatch-common", "andmatch-filtered", "andmatch-missing",
                    "andmatch-imdb", "andmatch-utf16", "andmatch-single-cjk", "andmatch-empty-filter", "andmatch-tv-flag",
                    "json-count-zero", "json-count-negative", "json-count-positive", "json-count-invalid", "json-count-missing", "html-count-ignored",
                    "rows-after", "dateheader-sibling", "dateheader-parent", "dateheader-missing-optional", "dateheader-missing-required",
                    "dateheader-invalid-nearest", "dateheader-nested-scope",
                    "query-flags-search", "query-flags-rss", "query-flags-year-zero", "query-flags-episode", "query-flags-music", "query-flags-book",
                    "query-flags-empty-imdb", "query-flags-empty-genre", "query-flags-all-ids",
                    "imdb-short-single-t", "imdb-short-repeated-t", "imdb-short-uppercase",
                    "checkbox-default-true", "checkbox-explicit-false", "checkbox-explicit-true", "select-default", "select-override",
                    "infohash-page", "infohash-before", "hash-only-public", "hash-only-semi-private", "hash-only-private", "query-parameter-case"}
        self.assertTrue(required.issubset({path.parent.name for path in cases}))
        for path in cases:
            with self.subTest(case=path.parent.name):
                resolved, expected, expected_errors = compare.resolve_case(path)
                self.assertIn("fixture.invalid", resolved["definition"])
                self.assertTrue(resolved["responses"])
                for difference in expected:
                    self.assertEqual(set(difference), {"path", "moji", "jackett", "reason"})
                    self.assertTrue(difference["path"].startswith("/"))
                    self.assertTrue(difference["reason"])
                    self.assertNotEqual(difference["moji"], difference["jackett"])


if __name__ == "__main__":
    unittest.main()

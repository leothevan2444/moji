"""Exercise version resolution against disposable Git repositories."""

import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


class VersionTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="moji-version-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "scripts").mkdir()
        self.script = self.root / "scripts/version.sh"
        shutil.copyfile(Path(__file__).with_name("version.sh"), self.script)
        (self.root / "source.txt").write_text("initial\n")
        (self.root / ".gitignore").write_text("ignored/\n")
        self.git("init", "-q")
        self.git("config", "user.name", "Version test")
        self.git("config", "user.email", "version-test@example.invalid")
        self.git("add", ".")
        self.git("commit", "-qm", "Initial source")
        self.sha = self.git("rev-parse", "--short=12", "HEAD").strip()

    def git(self, *args):
        return subprocess.check_output(
            ["git", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false", *args],
            cwd=self.root, text=True, stderr=subprocess.PIPE,
            env={**os.environ, "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_NOSYSTEM": "1"},
        )

    def resolve(self, check=True):
        result = subprocess.run(
            ["sh", str(self.script)], cwd="/tmp", text=True, capture_output=True,
        )
        if check:
            self.assertEqual(result.returncode, 0, result.stderr)
            return result.stdout.strip()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, "")
        return result.stderr

    def test_untagged_commit_is_development_version(self):
        self.assertEqual(self.resolve(), f"dev-{self.sha}")

    def test_lightweight_release_tag(self):
        self.git("tag", "v1.2.3")
        self.assertEqual(self.resolve(), "1.2.3")

    def test_annotated_release_tag(self):
        self.git("tag", "-a", "v0.2.0", "-m", "Release")
        self.assertEqual(self.resolve(), "0.2.0")

    def test_prerelease_tag(self):
        self.git("tag", "v1.0.0-rc.1")
        self.assertEqual(self.resolve(), "1.0.0-rc.1")

    def test_invalid_release_tags_fail(self):
        for tag in ("vnext", "v1.2", "v01.2.3", "v1.2.3-01", "v1.2.3-", "v1.2.3+build.1", "v1.2.3-" + "x" * 128):
            with self.subTest(tag=tag):
                self.git("tag", tag)
                self.assertIn("invalid release tag", self.resolve(check=False))
                self.git("tag", "-d", tag)

    def test_multiple_release_tags_are_ambiguous(self):
        self.git("tag", "v1.2.3")
        self.git("tag", "v2.0.0")
        self.assertIn("multiple v* tags", self.resolve(check=False))

    def test_nonrelease_tag_is_ignored(self):
        self.git("tag", "checkpoint")
        self.assertEqual(self.resolve(), f"dev-{self.sha}")

    def test_commit_after_release_does_not_reuse_version(self):
        self.git("tag", "v1.2.3")
        self.git("commit", "--allow-empty", "-qm", "Next change")
        sha = self.git("rev-parse", "--short=12", "HEAD").strip()
        self.assertEqual(self.resolve(), f"dev-{sha}")

    def test_modified_tagged_checkout_is_not_a_release(self):
        self.git("tag", "v1.2.3")
        (self.root / "source.txt").write_text("modified\n")
        self.assertEqual(self.resolve(), f"dev-{self.sha}-dirty")
        self.git("add", "source.txt")
        self.assertEqual(self.resolve(), f"dev-{self.sha}-dirty")

    def test_untracked_source_is_dirty(self):
        self.git("tag", "v1.2.3")
        (self.root / "extra.txt").write_text("new source\n")
        self.assertEqual(self.resolve(), f"dev-{self.sha}-dirty")

    def test_ignored_build_output_does_not_change_version(self):
        self.git("tag", "v1.2.3")
        (self.root / "ignored").mkdir()
        (self.root / "ignored/artifact").write_text("built\n")
        self.assertEqual(self.resolve(), "1.2.3")

    def test_missing_git_metadata_fails(self):
        shutil.rmtree(self.root / ".git")
        self.resolve(check=False)


if __name__ == "__main__":
    unittest.main()

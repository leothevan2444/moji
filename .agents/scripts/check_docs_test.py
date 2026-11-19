#!/usr/bin/env python3
"""Offline regression cases for document coverage and agent navigation."""

import contextlib
import importlib.util
import io
import sys
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch


sys.dont_write_bytecode = True  # Keep local checks from leaving untracked artifacts.

SPEC = importlib.util.spec_from_file_location("check_docs", Path(__file__).with_name("check_docs.py"))
docs = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(docs)


class DocumentationCheckTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory(prefix="moji-docs-test-")
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name).resolve()
        self.guides = self.root / ".agents"
        for name in ["AGENTS.md", "README.md", "architecture.md", "TODO.md",
                     "standards/product.md", "standards/engineering.md",
                     "workflows/development.md", "workflows/review.md", "modules/tasks.md"]:
            self.write(".agents/" + name, "# Guide\n")
        links = "\n".join(f"[guide]({p.relative_to(self.guides)})"
                          for p in sorted(self.guides.rglob("*.md")) if p.name != "AGENTS.md")
        self.write(".agents/AGENTS.md", "# Entry\n" + links)
        self.write("README.md", "# Moji\n")
        self.write("AGENTS.md", "[entry](.agents/AGENTS.md)\n")
        self.write(".github/copilot-instructions.md", "[entry](../.agents/AGENTS.md)\n")

    def write(self, name, content):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def check(self):
        output = io.StringIO()
        with patch.multiple(docs, ROOT=self.root, GUIDES=self.guides), contextlib.redirect_stdout(output):
            status = docs.main()
        return status, output.getvalue()

    def test_valid_local_anchors_and_fenced_examples(self):
        self.write("docs/topic.md", "# 标题\n# Repeat\n# Repeat\n<a id='explicit'></a>\n")
        self.write("README.md", "[unicode](docs/topic.md#标题)\n[repeat](docs/topic.md#repeat-1)\n"
                   "[explicit](docs/topic.md#explicit)\n```md\n[example](absent.md)\n```\n"
                   "[remote](https://example.invalid/never-requested)\n")
        status, output = self.check()
        self.assertEqual(status, 0, output)

    def test_user_docs_and_both_help_languages_are_checked(self):
        for name in ["README.md", "docs/setup.md", "web/src/help/zh/home.md", "web/src/help/en/home.md"]:
            with self.subTest(name=name):
                self.write(name, "[broken](missing.md)\n")
                status, output = self.check()
                self.assertEqual(status, 1)
                self.assertIn(name + ":1: missing", output)
                self.write(name, "# Valid\n")

    def test_existing_file_with_removed_anchor_is_rejected(self):
        self.write("docs/current.md", "# Current\n")
        self.write("README.md", "[stale](docs/current.md#removed)\n")
        status, output = self.check()
        self.assertEqual(status, 1)
        self.assertIn("missing anchor", output)

    def test_links_cannot_escape_repository(self):
        self.write("docs/setup.md", "[escape](../../outside.md)\n[absolute](/tmp/existing.md)\n")
        status, output = self.check()
        self.assertEqual(status, 1)
        self.assertIn("out-of-repository", output)
        self.assertIn("use a relative link", output)

    def test_orphaned_module_and_review_are_rejected(self):
        self.write(".agents/modules/new.md", "# New\n")
        self.write(".agents/reviews/orphan.md", "# Historical\n")
        status, output = self.check()
        self.assertEqual(status, 1)
        self.assertIn("modules guide not routed", output)
        self.assertIn("unreachable guide: .agents/reviews/orphan.md", output)

    def test_required_guides_and_forwarding_entries_are_enforced(self):
        (self.guides / "TODO.md").unlink()
        self.write("AGENTS.md", "# Detached entry\n")
        status, output = self.check()
        self.assertEqual(status, 1)
        self.assertIn("missing required guide: TODO.md", output)
        self.assertIn("entry must point to .agents/AGENTS.md: AGENTS.md", output)


if __name__ == "__main__":
    unittest.main()

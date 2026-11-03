"""Reject missing or changed dependency locks before reference acceptance."""

import copy
from pathlib import Path
import shutil
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
import build_reference as builder


class PackageLockTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.source = self.root / "source"
        shutil.copytree(builder.DIRECTORY / "nuget-locks", self.source / "nuget-locks")
        self.work = self.root / "work"
        for target in builder.PACKAGE_LOCK_TARGETS:
            (self.work / target).parent.mkdir(parents=True, exist_ok=True)
        self.pins = copy.deepcopy(builder.LOCK)
        self.directory_patch = patch.object(builder, "DIRECTORY", self.source)
        self.lock_patch = patch.object(builder, "LOCK", self.pins)
        self.directory_patch.start()
        self.lock_patch.start()
        self.addCleanup(self.directory_patch.stop)
        self.addCleanup(self.lock_patch.stop)

    def test_current_lock_set_is_complete_and_installs_exact_bytes(self):
        builder.install_package_locks(self.work)
        builder.verify_package_locks(self.work)

    def test_missing_and_changed_source_lock_cannot_be_installed(self):
        target = self.source / "nuget-locks" / builder.PACKAGE_LOCK_TARGETS[0]
        target.write_text("{}")
        with self.assertRaisesRegex(ValueError, "source lock checksum"):
            builder.install_package_locks(self.work)
        target.unlink()
        with self.assertRaisesRegex(ValueError, "source lock checksum"):
            builder.install_package_locks(self.work)

    def test_changed_or_removed_installed_lock_invalidates_reference(self):
        builder.install_package_locks(self.work)
        target = self.work / builder.PACKAGE_LOCK_TARGETS[0]
        target.write_text("{}")
        with self.assertRaisesRegex(ValueError, "installed lock checksum"):
            builder.verify_package_locks(self.work)
        target.unlink()
        with self.assertRaisesRegex(ValueError, "installed lock checksum"):
            builder.verify_package_locks(self.work)

    def test_unknown_target_cannot_escape_work_directory(self):
        self.pins["nugetLocks"]["../outside"] = "synthetic"
        with self.assertRaisesRegex(ValueError, "unknown NuGet lock targets"):
            builder.install_package_locks(self.work)
        self.assertFalse((self.root / "outside").exists())

    def test_missing_target_cannot_silently_leave_unlocked_project(self):
        del self.pins["nugetLocks"][builder.PACKAGE_LOCK_TARGETS[0]]
        with self.assertRaisesRegex(ValueError, "incomplete"):
            builder.install_package_locks(self.work)


if __name__ == "__main__":
    unittest.main()

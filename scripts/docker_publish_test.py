"""Verify that only the two tested images can reach release tags."""

import copy
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
import docker_publish as publisher


class PublishTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix="moji-publish-")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.image = "example/moji"
        self.version = "1.2.3"
        self.revision = "a" * 40
        self.calls = []
        self.images = {}
        self.digests = {}
        for arch, digit in (("amd64", "1"), ("arm64", "2")):
            self.images[arch] = {
                "Id": "sha256:" + digit * 64,
                "Os": "linux", "Architecture": arch,
                "Config": {"Labels": {
                    publisher.VERSION_LABEL: self.version,
                    publisher.REVISION_LABEL: self.revision,
                }},
            }
            self.digests[arch] = self.image + "@sha256:" + digit * 64
            directory = self.root / f"moji-image-{arch}"
            directory.mkdir()
            (directory / "image.tar").write_bytes(b"archive fixture")
            self.write_record(arch)
        self.loaded = copy.deepcopy(self.images)
        self.metadata = {"tags": [self.image + ":1.2.3", self.image + ":latest", self.image + ":sha-aaaaaaa"]}
        mocked = patch.object(publisher, "docker", side_effect=self.docker)
        mocked.start()
        self.addCleanup(mocked.stop)

    def write_record(self, arch):
        path = self.root / f"moji-image-{arch}" / "image.json"
        path.write_text(json.dumps([self.images[arch]]))

    def docker(self, *args, capture=False):
        self.calls.append(tuple(map(str, args)))
        if args[:2] == ("image", "inspect"):
            ref = args[2]
            if ref.startswith("moji-ci-"):
                arch = ref.removeprefix("moji-ci-").split(":", 1)[0]
                return json.dumps([self.loaded[arch]])
            arch = ref.rsplit("-", 1)[1]
            return json.dumps([{"RepoDigests": [self.digests[arch]]}])

    def publish(self):
        publisher.publish(self.root, self.image, self.version, self.revision, "123-2", self.metadata)

    def assert_no_push(self):
        self.assertFalse(any(call[:2] == ("image", "push") for call in self.calls))
        self.assertFalse(any(call[0] == "buildx" for call in self.calls))

    def test_publish_exact_image_ids_and_merge_verified_digests(self):
        self.publish()
        self.assertEqual(sum(call[:2] == ("image", "load") for call in self.calls), 2)
        for arch in publisher.ARCHITECTURES:
            self.assertIn(("image", "tag", self.images[arch]["Id"], self.image + f":ci-123-2-{arch}"), self.calls)
        merge = self.calls[-1]
        self.assertEqual(merge[:3], ("buildx", "imagetools", "create"))
        self.assertEqual(merge[-2:], (self.digests["amd64"], self.digests["arm64"]))
        push_index = next(i for i, call in enumerate(self.calls) if call[:2] == ("image", "push"))
        self.assertEqual(sum(call[:2] == ("image", "inspect") for call in self.calls[:push_index]), 2)

    def test_missing_second_architecture_stops_before_loading_or_pushing(self):
        (self.root / "moji-image-arm64" / "image.tar").unlink()
        with self.assertRaisesRegex(ValueError, "Missing tested linux/arm64"):
            self.publish()
        self.assertEqual(self.calls, [])

    def test_wrong_record_platform_or_identity_stops_before_docker(self):
        for field, value in (("Architecture", "amd64"), ("Os", "windows"), ("Id", "not-a-digest")):
            with self.subTest(field=field):
                saved = self.images["arm64"][field]
                self.images["arm64"][field] = value
                self.write_record("arm64")
                with self.assertRaises(ValueError):
                    self.publish()
                self.assertEqual(self.calls, [])
                self.images["arm64"][field] = saved

    def test_wrong_record_version_or_revision_stops_before_docker(self):
        for key in (publisher.VERSION_LABEL, publisher.REVISION_LABEL):
            with self.subTest(label=key):
                labels = self.images["arm64"]["Config"]["Labels"]
                saved = labels[key]
                labels[key] = "other"
                self.write_record("arm64")
                with self.assertRaisesRegex(ValueError, "does not match"):
                    self.publish()
                self.assertEqual(self.calls, [])
                labels[key] = saved

    def test_loaded_image_must_match_post_test_image_id(self):
        self.loaded["arm64"]["Id"] = "sha256:" + "3" * 64
        with self.assertRaisesRegex(ValueError, "differs from the tested image"):
            self.publish()
        self.assert_no_push()

    def test_loaded_image_must_have_expected_architecture(self):
        self.loaded["arm64"]["Architecture"] = "amd64"
        with self.assertRaisesRegex(ValueError, "Expected linux/arm64"):
            self.publish()
        self.assert_no_push()

    def test_loaded_image_must_have_expected_revision(self):
        self.loaded["arm64"]["Config"]["Labels"][publisher.REVISION_LABEL] = "other"
        with self.assertRaisesRegex(ValueError, "does not match"):
            self.publish()
        self.assert_no_push()

    def test_failed_second_push_never_updates_release_tags(self):
        original = self.docker

        def fail_push(*args, **kwargs):
            if args[:2] == ("image", "push") and args[2].endswith("-arm64"):
                raise subprocess.CalledProcessError(1, ["docker", *args])
            return original(*args, **kwargs)

        with patch.object(publisher, "docker", side_effect=fail_push):
            with self.assertRaises(subprocess.CalledProcessError):
                self.publish()
        self.assertFalse(any(call[0] == "buildx" for call in self.calls))

    def test_missing_or_foreign_push_digest_never_updates_release_tags(self):
        self.digests["arm64"] = "other/moji@sha256:" + "2" * 64
        with self.assertRaisesRegex(ValueError, "Expected one pushed linux/arm64 digest"):
            self.publish()
        self.assertFalse(any(call[0] == "buildx" for call in self.calls))

    def test_metadata_cannot_redirect_release_to_another_repository(self):
        for tags in ([], ["other/moji:latest"]):
            with self.subTest(tags=tags):
                self.metadata["tags"] = tags
                with self.assertRaisesRegex(ValueError, "configured image repository"):
                    self.publish()
                self.assertEqual(self.calls, [])

    def test_metadata_cannot_change_release_version(self):
        self.metadata["labels"] = {publisher.VERSION_LABEL: "other"}
        with self.assertRaisesRegex(ValueError, "Release metadata does not match"):
            self.publish()
        self.assertEqual(self.calls, [])

    def test_preserves_metadata_tags_for_edge_stable_and_prerelease(self):
        for names in (("edge", "sha-aaaaaaa"), ("1.2.3", "latest"), ("1.3.0-rc.1",)):
            with self.subTest(names=names):
                self.calls.clear()
                self.metadata["tags"] = [self.image + ":" + name for name in names]
                self.publish()
                merge = self.calls[-1]
                actual = [merge[i + 1] for i, argument in enumerate(merge) if argument == "--tag"]
                self.assertEqual(actual, self.metadata["tags"])


if __name__ == "__main__":
    unittest.main()

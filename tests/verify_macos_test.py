"""签名验证器负例；mock 结果不作为 Apple 或 macOS 平台验收证据。"""

import importlib.util
import unittest
from pathlib import Path

spec = importlib.util.spec_from_file_location("verify_macos", Path(__file__).resolve().parents[1] / "scripts/verify-macos.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class SignatureTests(unittest.TestCase):
    def check(self, detail=None, assessment="source=Notarized Developer ID"):
        outputs = iter(["", detail if detail is not None else (
            "TeamIdentifier=ABCDEFGHIJ\nAuthority=Developer ID Application: Test\n"
            "flags=0x10000(runtime)\nTimestamp=Sep 14, 2026\n"
        ), assessment])
        module.verify_signature(Path("synthetic-binary"), "ABCDEFGHIJ", execute=lambda args: next(outputs))

    def test_verified_signature(self):
        self.check()

    def test_wrong_team_or_adhoc_is_rejected(self):
        for detail in ["TeamIdentifier=OTHERTEAM1", "TeamIdentifier=ABCDEFGHIJ\nSignature=adhoc"]:
            with self.subTest(detail=detail), self.assertRaises(ValueError):
                self.check(detail=detail)

    def test_missing_runtime_or_timestamp_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "runtime"):
            self.check(detail="TeamIdentifier=ABCDEFGHIJ\nAuthority=Developer ID Application: Test\n")

    def test_non_notarized_gatekeeper_result_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "公证"):
            self.check(assessment="source=Developer ID")


if __name__ == "__main__":
    unittest.main()

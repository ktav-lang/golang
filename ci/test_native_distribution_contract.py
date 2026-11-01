"""Проверить обязательные combined targets и distribution gates в CI."""

from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
FILES = ['ci/check_native_distribution.rb', '.github/workflows/native-distribution.yml', '.github/workflows/windows-gnu-cabi-candidate.yml', '.github/workflows/native-distribution-check.yml', '.github/workflows/release.yml', 'scripts/stage_go_gnu_release.py']
GNU = '.github/workflows/windows-gnu-cabi-candidate.yml'
NATIVE = '.github/workflows/native-distribution.yml'


class NativeDistributionContractTests(unittest.TestCase):
    def check(self, path=None, before=None, after=None):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for name in FILES:
                destination = root / name
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(ROOT / name, destination)
            if path is not None:
                destination = root / path
                text = destination.read_text(encoding='utf-8')
                self.assertEqual(text.count(before), 1, 'unique actual workflow mutation anchor')
                destination.write_text(text.replace(before, after), encoding='utf-8')
            return subprocess.run(['ruby', 'ci/check_native_distribution.rb'], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, timeout=30)

    def reject(self, path, before, after, reason):
        result = self.check(path, before, after)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn(reason, result.stderr)

    def test_actual_workflow_preserves_all_six_native_roles(self):
        result = self.check()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('native distribution workflow and publication gates verified', result.stdout)

    def test_missing_genuine_gnu_target_is_rejected(self):
        self.reject(GNU, '          - target: aarch64-pc-windows-gnullvm\n', '          - target: aarch64-pc-windows-msvc\n', 'wrong GNU target matrix')

    def test_missing_source_forwarding_is_rejected(self):
        self.reject(NATIVE, '      source-ref: ${{ inputs.source-ref }}\n', '      source-ref: fixed-unreviewed-source\n', 'GNU source is not forwarded')

    def test_missing_consumer_success_gate_is_rejected(self):
        self.reject(GNU, '    needs: [cabi-gnullvm, go-consumers]\n', '    needs: [cabi-gnullvm]\n', 'missing complete GNU consumer gate')

    def test_wrong_arm_runner_is_rejected(self):
        self.reject(GNU, '          - os: windows-11-arm\n', '          - os: windows-latest\n', 'incomplete genuine Go consumer matrix')

    def test_missing_sidecar_role_is_rejected(self):
        self.reject('scripts/stage_go_gnu_release.py', "asset.name + '.NOTICE.manifest.json'", "asset.name + '.unexpected.json'", 'missing GNU delivery role .NOTICE.manifest.json')

    def test_missing_publication_dependency_is_rejected(self):
        self.reject('.github/workflows/release.yml', '    needs: build-native\n', '    needs: missing-native\n', 'missing native publication gate')

    def test_broad_release_artifact_selector_is_rejected(self):
        self.reject('.github/workflows/release.yml', '          pattern: native-distribution-*\n', '          pattern: native-*\n', 'unsafe release artifact download')


if __name__ == '__main__':
    unittest.main()

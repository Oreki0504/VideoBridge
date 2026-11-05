import os
from pathlib import Path
import shutil
import subprocess
import unittest


ROOT = Path(__file__).resolve().parents[1]
BASH = os.environ.get('VB_TEST_BASH') or (
    r'C:\Program Files\Git\bin\bash.exe' if os.name == 'nt' else shutil.which('bash')
)
SCRIPTS = ('Install.command', 'Uninstall.command', 'Build-Pkg.command')


class MacScriptTests(unittest.TestCase):
    def test_syntax(self):
        for name in SCRIPTS:
            with self.subTest(script=name):
                result = subprocess.run(
                    [BASH, '--noprofile', '--norc', '-n', str(ROOT / 'installer/macos' / name)],
                    capture_output=True, text=True, encoding='utf-8',
                )
                self.assertEqual(result.returncode, 0, result.stderr)

    def test_language_selection(self):
        cases = (
            ('system_chinese', {}, '(\n    "zh-Hans-CN",\n    en\n)', [], 'zh'),
            ('system_unquoted_chinese', {}, '(\n    zh_CN,\n    en\n)', [], 'zh'),
            ('system_english_first', {}, '(\n    en,\n    "zh-Hans-CN"\n)', [], 'en'),
            ('system_english_region', {}, '(\n    "en-GB",\n    "zh-Hans"\n)', [], 'en'),
            ('system_unknown', {}, '', [], 'en'),
            ('terminal_chinese', {'LANG': 'zh_TW.UTF-8'}, '', [], 'zh'),
            ('terminal_precedence', {'LC_ALL': 'en_US.UTF-8', 'LANG': 'zh_CN.UTF-8'}, '', [], 'en'),
            ('environment_override', {'VIDEOBRIDGE_LANG': 'en', 'LANG': 'zh_CN.UTF-8'}, '', [], 'en'),
            ('argument_english', {'VIDEOBRIDGE_LANG': 'zh'}, '', ['--lang=en'], 'en'),
            ('argument_chinese', {'VIDEOBRIDGE_LANG': 'en'}, '', ['--lang=zh'], 'zh'),
        )
        for name in SCRIPTS:
            source = (ROOT / 'installer/macos' / name).read_text(encoding='utf-8')
            marker = 'export VIDEOBRIDGE_LANG="$(detect_lang)"'
            self.assertIn(marker, source)
            selection = source.split(marker, 1)[0] + marker
            harness = '''
function uname { printf 'Darwin\\n'; }
function /usr/bin/defaults { printf '%s\\n' "$VB_TEST_APPLE_LANGUAGES"; }
''' + selection + '\nprintf "%s" "$VIDEOBRIDGE_LANG"\n'
            for label, overrides, apple_languages, arguments, expected in cases:
                with self.subTest(script=name, case=label):
                    environment = {**os.environ, 'LANG': 'C', 'LC_ALL': '', 'LC_MESSAGES': '',
                                   'VIDEOBRIDGE_LANG': '', 'VB_TEST_APPLE_LANGUAGES': apple_languages,
                                   **overrides}
                    result = subprocess.run(
                        [BASH, '--noprofile', '--norc', '-c', harness, name, *arguments],
                        env=environment, capture_output=True, text=True, encoding='utf-8',
                    )
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertNotIn('sed:', result.stderr)
                    self.assertEqual(result.stdout, expected, result.stderr)


if __name__ == '__main__':
    unittest.main()

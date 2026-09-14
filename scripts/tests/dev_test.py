"""Local F5 preparation: configuration reuse, port collisions and failure handling."""
import importlib.util
import json
import os
from pathlib import Path
import socket
import tempfile
import tomllib
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('castor_dev', ROOT / 'scripts/dev.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class DevTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='castor-dev-unit-')
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.dev = module.DevEnvironment(self.root)

    def test_settings_are_created_once_and_keep_credentials(self):
        with patch.object(self.dev, 'say'):
            first = self.dev.settings()
            again = self.dev.settings()
        self.assertEqual(first, again)
        self.assertEqual(len(first['rsa_secret']), 32)
        self.assertGreaterEqual(len(first['admin_password']), 8)
        self.assertEqual((self.dev.state / 'settings.json').stat().st_mode & 0o777, 0o600)

    def test_port_conflict_uses_an_available_port(self):
        with socket.socket() as occupied:
            occupied.bind(('127.0.0.1', 0))
            port = occupied.getsockname()[1]
            chosen = self.dev.available_port(port)
            self.assertNotEqual(chosen, port)
            with socket.socket() as probe:
                probe.bind(('127.0.0.1', chosen))

    def test_toml_keeps_custom_options_and_escapes_values(self):
        source = '[General]\nJwtKey = "old"\nRegisterEnabled = false\n'
        output = self.dev.configure_toml(source, {('General', 'JwtKey'): 'a"b\\c'})
        parsed = tomllib.loads(output)
        self.assertEqual(parsed['General']['JwtKey'], 'a"b\\c')
        self.assertFalse(parsed['General']['RegisterEnabled'])

    def test_invalid_jwt_lifetimes_are_reset_but_valid_custom_values_kept(self):
        stale = '[General]\nJwtTimeoutHours = 168\nJwtMaxRefreshHours = 24\n'
        fixed = tomllib.loads(self.dev.configure_toml(stale, self.dev.jwt_lifetime_fixes(stale)))
        self.assertEqual((fixed['General']['JwtTimeoutHours'], fixed['General']['JwtMaxRefreshHours']), (2, 168))
        custom = '[General]\nJwtTimeoutHours = 4\nJwtMaxRefreshHours = 72\n'
        self.assertEqual(self.dev.jwt_lifetime_fixes(custom), {})

    def test_missing_managed_field_fails_instead_of_silently_misconfiguring(self):
        with self.assertRaises(ValueError):
            self.dev.configure_toml('[General]\n', {('General', 'JwtKey'): 'test'})

    def test_missing_tools_fail_before_creating_infrastructure(self):
        with patch.object(module.shutil, 'which', return_value=None), patch.object(self.dev, 'ensure_docker') as docker:
            with self.assertRaises(RuntimeError):
                self.dev.prepare()
            docker.assert_not_called()
            self.assertFalse(self.dev.state.exists())

    def test_compose_uses_dedicated_project_and_overrides_inherited_credentials(self):
        with patch.object(self.dev, 'say'):
            settings = self.dev.settings()
        with patch.dict(os.environ, {'POSTGRES_PASSWORD': 'unrelated-password'}):
            env = self.dev.compose_env(settings)
        self.assertEqual(env['POSTGRES_PASSWORD'], settings['postgres_password'])
        self.assertTrue(env['PROJECT_NAME'].startswith('castor-dev-'))
        self.assertEqual(env['POSTGRES_PORT'], '0')
        self.assertIn(str(self.dev.state / 'compose.env'), self.dev.compose())

    def test_stop_without_state_does_not_create_services(self):
        with patch.object(self.dev, 'run') as run:
            self.dev.stop()
            run.assert_not_called()
        self.assertFalse(self.dev.state.exists())

    def test_launch_prepares_then_waits_for_api_and_stops_compound(self):
        launch = json.loads((ROOT / '.vscode/launch.json').read_text())
        compound = launch['compounds'][0]
        self.assertEqual(compound['name'], 'Castor')
        self.assertEqual(compound['preLaunchTask'], 'Castor: Prepare')
        self.assertTrue(compound['stopAll'])
        web = next(c for c in launch['configurations'] if c['name'] == 'Castor Web')
        self.assertEqual(web['preLaunchTask'], 'Castor: Wait API')
        import re
        self.assertEqual(re.search(web['serverReadyAction']['pattern'], '- Local: http://127.0.0.1:3000').group(1), 'http://127.0.0.1:3000')

    def test_all_languages_have_matching_messages(self):
        messages = json.loads((ROOT / 'scripts/i18n/dev.json').read_text())
        for language in ('zh', 'en', 'ja', 'ko'):
            self.assertEqual(set(messages[language]), set(messages['en']))
            self.assertTrue(all(messages[language].values()))


if __name__ == '__main__':
    unittest.main()

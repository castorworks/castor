"""Local F5 preparation: configuration reuse, port collisions and failure handling."""
import importlib.util
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import tomllib
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('castor_dev', ROOT / 'scripts/dev.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class DevTest(unittest.TestCase):
    def test_old_python_reports_requirement_before_importing_tomllib(self):
        messages = json.loads((ROOT / 'scripts/i18n/dev.json').read_text(encoding='utf-8'))
        code = '''
import runpy, sys
sys.version_info = (3, 8, 10)
sys.modules['tomllib'] = None
runpy.run_path(sys.argv[1], run_name='__main__')
'''
        # 一种语言即可：各语言文案齐全由 test_all_languages_have_matching_messages 保证
        result = subprocess.run(
            [sys.executable, '-c', code, str(ROOT / 'scripts/dev.py')],
            env=dict(os.environ, CASTOR_DEV_LANG='en', PYTHONIOENCODING='utf-8'),
            capture_output=True, text=True, encoding='utf-8')
        self.assertEqual(result.returncode, 1)
        self.assertNotIn('Traceback', result.stderr)
        self.assertIn(messages['en']['python_version'].format(
            version='3.8.10', executable=sys.executable), result.stderr)

    def test_windows_tasks_use_python_launcher(self):
        tasks = json.loads((ROOT / '.vscode/tasks.json').read_text(encoding='utf-8'))['tasks']
        for task in tasks:
            with self.subTest(task=task['label']):
                self.assertEqual(task['command'], 'python3')
                self.assertEqual(task['windows']['command'], 'py')
                self.assertEqual(task['windows']['args'], ['-3'] + task['args'])

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
        if os.name == 'posix':  # NTFS permissions are not expressed as mode bits
            self.assertEqual((self.dev.state / 'settings.json').stat().st_mode & 0o777, 0o600)

    def test_port_conflict_uses_an_available_port(self):
        with socket.socket() as occupied:
            occupied.bind(('127.0.0.1', 0))
            port = occupied.getsockname()[1]
            chosen = self.dev.available_port(port)
            self.assertNotEqual(chosen, port)
            with socket.socket() as probe:
                probe.bind(('127.0.0.1', chosen))

    def test_recently_stopped_server_keeps_its_port(self):
        # Stopping a dev server leaves its closed connections in TIME_WAIT; the next
        # `prepare` must still reuse the port instead of moving the app elsewhere.
        with socket.socket() as server:
            # Like Node and Go listeners: on Linux a TIME_WAIT connection only lets the
            # port be rebound when the original listener also set SO_REUSEADDR.
            if os.name != 'nt':
                server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            server.bind(('127.0.0.1', 0))
            server.listen()
            port = server.getsockname()[1]
            with socket.create_connection(('127.0.0.1', port)) as client:
                accepted, _ = server.accept()
                accepted.close()  # server side closes first and holds the TIME_WAIT
                client.recv(1)
        self.assertEqual(self.dev.available_port(port), port)

    def test_toml_keeps_custom_options_and_escapes_values(self):
        source = '[General]\nJwtKey = "old"\nRegisterEnabled = false\n'
        output = self.dev.configure_toml(source, {('General', 'JwtKey'): 'a"b\\c'})
        parsed = tomllib.loads(output)
        self.assertEqual(parsed['General']['JwtKey'], 'a"b\\c')
        self.assertFalse(parsed['General']['RegisterEnabled'])

    def test_config_prepared_before_a_field_existed_gets_it_from_the_template(self):
        local = '[General]\nJwtKey = "old"\nCustom = 1\n\n[Postgres]\nHost = "db"\n'
        template = '[General]\nJwtKey = ""\nPublicURL = ""\n\n[Security]\nDataEncryptionKey = ""\n'
        keys = [('General', 'PublicURL'), ('Security', 'DataEncryptionKey'), ('General', 'JwtKey')]
        merged = self.dev.add_template_fields(local, template, keys)
        output = self.dev.configure_toml(merged, {
            ('General', 'PublicURL'): 'http://127.0.0.1:3000', ('Security', 'DataEncryptionKey'): 'k' * 64,
            ('General', 'JwtKey'): 'new'})
        parsed = tomllib.loads(output)
        self.assertEqual(parsed['General'], {'JwtKey': 'new', 'Custom': 1, 'PublicURL': 'http://127.0.0.1:3000'})
        self.assertEqual(parsed['Security']['DataEncryptionKey'], 'k' * 64)
        self.assertEqual(parsed['Postgres']['Host'], 'db')
        # A field the template does not have still fails loudly.
        with self.assertRaises(ValueError):
            self.dev.configure_toml(self.dev.add_template_fields(local, template, [('General', 'Nope')]),
                                    {('General', 'Nope'): 1})

    def test_example_mail_host_is_recognized_like_the_api_does(self):
        # 旧模板留下的 smtp.example.com 会让 API 拒绝启动；prepare 据此把它清空。
        for host, expected in {'smtp.example.com': True, 'Relay.Example.Org.': True, 'mail.example': True,
                               '': False, 'smtp.gmail.com': False, 'example.com.cn': False}.items():
            self.assertEqual(self.dev.example_mail_host(f'[Mail]\nHost = "{host}"\n'), expected, host)
        self.assertFalse(self.dev.example_mail_host('[General]\n'))

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
        launch = json.loads((ROOT / '.vscode/launch.json').read_text(encoding='utf-8'))
        compound = launch['compounds'][0]
        self.assertEqual(compound['name'], 'Castor')
        self.assertEqual(compound['preLaunchTask'], 'Castor: Prepare')
        self.assertTrue(compound['stopAll'])
        web = next(c for c in launch['configurations'] if c['name'] == 'Castor Web')
        self.assertEqual(web['preLaunchTask'], 'Castor: Wait API')
        import re
        self.assertEqual(re.search(web['serverReadyAction']['pattern'], '- Local: http://127.0.0.1:3000').group(1), 'http://127.0.0.1:3000')

    def test_claude_launch_follows_the_selected_ports_and_holds_no_secrets(self):
        with patch.object(self.dev, 'say'):
            settings = self.dev.settings()
        settings['api_port'], settings['web_port'] = 45001, 45002
        self.dev.write_claude_launch(settings)

        path = self.root / '.claude/launch.json'
        launch = json.loads(path.read_text(encoding='utf-8'))
        by_name = {c['name']: c for c in launch['configurations']}
        # 端口写错时 preview 只会打开一张空白页，不会报错，所以必须跟随实际选定的端口。
        self.assertEqual(by_name['castor-api']['port'], 45001)
        self.assertEqual(by_name['castor-web']['port'], 45002)

        # 不经过 sh：Windows 上同一份配置也能直接启动。
        for configuration in by_name.values():
            self.assertEqual(configuration['runtimeExecutable'], sys.executable)
            self.assertEqual(configuration['runtimeArgs'][0], str(self.root / 'scripts/dev.py'))
        self.assertEqual(by_name['castor-api']['runtimeArgs'][1:], ['run', 'api'])
        self.assertEqual(by_name['castor-web']['runtimeArgs'][1:], ['run', 'web'])

        # 凭据只能留在 debug.env 里；这个文件经 dev.py run 读取它，不复制它。
        text = path.read_text(encoding='utf-8')
        for secret in ('jwt_key', 'rsa_secret', 'admin_password', 'postgres_password'):
            self.assertNotIn(settings[secret], text)

    def test_run_starts_the_server_with_debug_env_instead_of_inherited_values(self):
        self.dev.state.mkdir(parents=True)
        self.dev.write_env(self.dev.state / 'debug.env', {
            'CASTOR_DB_HOST': '127.0.0.1', 'CASTOR_CONFIG_FILE': 'C:/castor/.local/dev/api.config.toml'})
        started = {}

        def execve(path, args, env):
            started.update(args=args, env=env)

        def run(args, cwd, env):
            started.update(args=args, env=env, cwd=cwd)
            return subprocess.CompletedProcess(args, 0)

        with patch.dict(os.environ, {'CASTOR_DB_NAME': 'production'}), \
                patch.object(module.os, 'execve', execve), patch.object(module.os, 'chdir'), \
                patch.object(module.subprocess, 'run', run):
            self.dev.start('api')
        self.assertEqual(started['args'][1:], ['run', './cmd/castor'])
        self.assertEqual(started['env']['CASTOR_DB_HOST'], '127.0.0.1')
        self.assertEqual(started['env']['CASTOR_CONFIG_FILE'], 'C:/castor/.local/dev/api.config.toml')
        self.assertNotIn('CASTOR_DB_NAME', started['env'])

    def test_run_before_prepare_explains_what_to_do(self):
        with self.assertRaises(RuntimeError):
            self.dev.start('web')

    def test_claude_launch_is_ignored_by_git(self):
        # 它固定了本机端口，提交上去会把一个人的端口分发给所有人。
        ignored = (ROOT / '.gitignore').read_text(encoding='utf-8').splitlines()
        self.assertIn('.claude/launch.json', ignored)

    def test_all_languages_have_matching_messages(self):
        messages = json.loads((ROOT / 'scripts/i18n/dev.json').read_text(encoding='utf-8'))
        for language in ('zh', 'en', 'ja', 'ko'):
            self.assertEqual(set(messages[language]), set(messages['en']))
            self.assertTrue(all(messages[language].values()))


if __name__ == '__main__':
    unittest.main()

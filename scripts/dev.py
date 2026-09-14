#!/usr/bin/env python3
"""Prepare the local VS Code environment; Python 3.11+, Docker, Go, Node and Bun."""
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import socket
import subprocess
import sys
import time
import tomllib
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]


class DevEnvironment:
    def __init__(self, root=ROOT):
        self.root = Path(root)
        self.state = self.root / '.local/dev'
        language = os.environ.get('CASTOR_DEV_LANG', os.environ.get('LANG', 'zh')).split('_')[0].split('-')[0]
        self.language = language if language in ('en', 'zh', 'ja', 'ko') else 'zh'
        self.messages = json.loads((ROOT / 'scripts/i18n/dev.json').read_text())[self.language]

    def message(self, key, **values):
        return self.messages[key].format(**values)

    def say(self, key, **values):
        print(self.message(key, **values), flush=True)

    def run(self, args, **kwargs):
        return subprocess.run(args, check=True, **kwargs)

    def write_private(self, path, value):
        path.write_text(value, encoding='utf-8')
        path.chmod(0o600)

    def settings(self):
        self.state.mkdir(parents=True, exist_ok=True, mode=0o700)
        path = self.state / 'settings.json'
        if path.exists():
            return json.loads(path.read_text())
        values = {
            'project': 'castor-dev-' + hashlib.sha256(str(self.root).encode()).hexdigest()[:8],
            'api_port': 1234, 'web_port': 3000,
            'jwt_key': secrets.token_hex(32), 'rsa_secret': secrets.token_hex(16),
            'admin_password': secrets.token_urlsafe(18),
            'postgres_password': secrets.token_hex(24),
            'redis_password': secrets.token_hex(24), 's3_secret': secrets.token_hex(24),
        }
        self.write_private(path, json.dumps(values, indent=2) + '\n')
        self.say('created', path=path)
        return values

    @staticmethod
    def available_port(preferred, excluded=()):
        with socket.socket() as sock:
            try:
                if preferred in excluded:
                    raise OSError()
                sock.bind(('127.0.0.1', preferred))
            except OSError:
                sock.bind(('127.0.0.1', 0))
            return sock.getsockname()[1]

    def ensure_docker(self):
        def ready():
            try:
                return subprocess.run(['docker', 'info'], stdout=subprocess.DEVNULL,
                                      stderr=subprocess.DEVNULL, timeout=10).returncode == 0
            except subprocess.TimeoutExpired:
                return False
        if ready():
            return
        self.say('docker_start')
        if sys.platform == 'darwin':
            self.run(['open', '-a', 'Docker'])
        else:
            # Docker Desktop CLI works on Windows/WSL and Desktop-enabled Linux.
            result = subprocess.run(['docker', 'desktop', 'start'], timeout=90)
            if result.returncode:
                raise RuntimeError(self.message('docker_manual'))
        deadline = time.monotonic() + 120
        while time.monotonic() < deadline:
            if ready():
                return
            time.sleep(2)
        raise RuntimeError(self.message('docker_manual'))

    def compose(self):
        directory = self.root / 'deploy/compose'
        return ['docker', 'compose', '--project-directory', str(directory),
                '--env-file', str(self.state / 'compose.env'),
                '-f', str(directory / 'compose.infra.yaml'),
                '-f', str(directory / 'compose.dev.yaml')]

    def compose_env(self, settings):
        values = {
            'PROJECT_NAME': settings['project'], 'POSTGRES_USER': 'castor',
            'POSTGRES_DB': 'castor', 'POSTGRES_PASSWORD': settings['postgres_password'],
            'REDIS_PASSWORD': settings['redis_password'], 'S3_ACCESS_KEY': 'castor',
            'S3_SECRET_KEY': settings['s3_secret'], 'S3_BUCKET': 'castor',
            'POSTGRES_PORT': '0', 'REDIS_PORT': '0',
            'MINIO_API_PORT': '0', 'MINIO_CONSOLE_PORT': '0',
        }
        # Values are passed directly to the child process, so shell syntax is never evaluated.
        self.write_private(self.state / 'compose.env', '\n'.join(
            f'{key}={json.dumps(value)}' for key, value in values.items()) + '\n')
        return dict(os.environ, **values)

    def published_port(self, service, port, env):
        address = self.run(self.compose() + ['port', service, str(port)],
                           env=env, capture_output=True, text=True).stdout.strip()
        return int(address.rsplit(':', 1)[1])

    @staticmethod
    def configure_toml(text, values):
        section = ''
        lines = []
        remaining = set(values)
        for line in text.splitlines():
            header = re.match(r'^\[([^]]+)\]', line)
            if header:
                section = header.group(1)
            field = re.match(r'^(\w+)\s*=', line)
            key = (section, field.group(1)) if field else None
            if key in values:
                line = field.group(1) + ' = ' + json.dumps(values[key], ensure_ascii=False)
                remaining.discard(key)
            lines.append(line)
        if remaining:
            raise ValueError(', '.join('.'.join(key) for key in sorted(remaining)))
        result = '\n'.join(lines) + '\n'
        tomllib.loads(result)
        return result

    @staticmethod
    def jwt_lifetime_fixes(text):
        """旧开发配置的 JWT 时长不满足 API 校验时，恢复为示例默认值；合法的自定义值保持不变。"""
        general = tomllib.loads(text).get('General', {})
        timeout = general.get('JwtTimeoutHours', 0)
        max_refresh = general.get('JwtMaxRefreshHours', 0)
        if 0 < timeout <= max_refresh:
            return {}
        return {('General', 'JwtTimeoutHours'): 2, ('General', 'JwtMaxRefreshHours'): 168}

    def prepare(self):
        for command in ('docker', 'go', 'node', 'bun'):
            if not shutil.which(command):
                raise RuntimeError(self.message('missing', command=command))
        self.ensure_docker()
        settings = self.settings()
        settings['api_port'] = self.available_port(settings['api_port'])
        settings['web_port'] = self.available_port(settings['web_port'], (settings['api_port'],))
        self.write_private(self.state / 'settings.json', json.dumps(settings, indent=2) + '\n')
        env = self.compose_env(settings)
        self.say('infrastructure')
        self.run(self.compose() + ['up', '-d', '--wait', '--wait-timeout', '180',
                                  'postgres', 'redis', 'minio'], env=env)
        self.run(self.compose() + ['run', '--rm', '--no-deps', 'minio-init'], env=env)
        ports = {service: self.published_port(service, port, env)
                 for service, port in [('postgres', 5432), ('redis', 6379), ('minio', 9000)]}
        api_url = f'http://127.0.0.1:{settings["api_port"]}'
        web_url = f'http://127.0.0.1:{settings["web_port"]}'
        config_path = self.state / 'api.config.toml'
        source = config_path if config_path.exists() else self.root / 'deploy/config/api.config.example.toml'
        values = {
            ('General', 'Address'): f'127.0.0.1:{settings["api_port"]}',
            ('General', 'Development'): True,
            ('General', 'JwtKey'): settings['jwt_key'],
            ('Security', 'RSAPrivateKeySecret'): settings['rsa_secret'],
            ('CORS', 'AllowOrigins'): [web_url, f'http://localhost:{settings["web_port"]}'],
            ('Postgres', 'Host'): '127.0.0.1', ('Postgres', 'Port'): ports['postgres'],
            ('Postgres', 'Username'): 'castor', ('Postgres', 'DbName'): 'castor',
            ('Postgres', 'Password'): settings['postgres_password'],
            ('Redis', 'Address'): f'127.0.0.1:{ports["redis"]}',
            ('Redis', 'Password'): settings['redis_password'],
            ('S3', 'Endpoint'): f'127.0.0.1:{ports["minio"]}',
            ('S3', 'AccessKey'): 'castor', ('S3', 'Secret'): settings['s3_secret'],
            ('S3', 'Bucket'): 'castor',
        }
        source_text = source.read_text()
        values.update(self.jwt_lifetime_fixes(source_text))
        self.write_private(config_path, self.configure_toml(source_text, values))
        self.say('dependencies')
        web = self.root / 'apps/web'
        fingerprint = hashlib.sha256((web / 'package.json').read_bytes() + (web / 'bun.lock').read_bytes()).hexdigest()
        stamp = self.state / 'web-dependencies.sha256'
        if not (web / 'node_modules/next/dist/bin/next').exists() or not stamp.exists() or stamp.read_text() != fingerprint:
            self.run(['bun', 'install', '--no-save', '--frozen-lockfile'], cwd=web)
            # Old Turbopack output may refer to modules from a previous dependency tree.
            # Refresh only the generated development cache on first setup/dependency changes.
            if (web / '.next/dev').exists():
                shutil.rmtree(web / '.next/dev')
            self.write_private(stamp, fingerprint)
        self.run(['go', 'mod', 'download'], cwd=self.root / 'apps/api')
        # Clear inherited CASTOR_* values to keep this dedicated environment self-contained.
        runtime = {key: value for key, value in os.environ.items() if not key.startswith('CASTOR_')}
        runtime.update(CASTOR_CONFIG_FILE=str(config_path),
                       CASTOR_DEFAULT_ADMIN_PASSWORD=settings['admin_password'])
        self.say('initialize')
        self.run(['go', 'run', './cmd/castor', 'init-db'], cwd=self.root / 'apps/api', env=runtime)
        debug_env = {
            'CASTOR_CONFIG_FILE': str(config_path), 'CASTOR_API_URL': api_url,
            'PORT': str(settings['web_port']), 'NODE_ENV': 'development',
            'NEXT_PUBLIC_API_URL': '', 'NEXT_PUBLIC_SENTRY_DISABLED': 'true',
            'BUILD_STANDALONE': 'false',
        }
        # Explicitly supply API overrides so inherited terminal variables cannot change the target DB.
        debug_env.update({
            'CASTOR_JWT_KEY': settings['jwt_key'], 'CASTOR_RSA_SECRET': settings['rsa_secret'],
            'CASTOR_DB_HOST': '127.0.0.1', 'CASTOR_DB_PORT': str(ports['postgres']),
            'CASTOR_DB_USER': 'castor', 'CASTOR_DB_NAME': 'castor',
            'CASTOR_DB_PASSWORD': settings['postgres_password'],
            'CASTOR_REDIS_ADDR': f'127.0.0.1:{ports["redis"]}',
            'CASTOR_REDIS_PASSWORD': settings['redis_password'],
            'CASTOR_S3_ENDPOINT': f'127.0.0.1:{ports["minio"]}',
            'CASTOR_S3_ACCESS_KEY': 'castor', 'CASTOR_S3_SECRET_KEY': settings['s3_secret'],
            'CASTOR_S3_BUCKET': 'castor', 'CASTOR_CORS_ORIGINS': web_url,
        })
        self.write_private(self.state / 'debug.env', '\n'.join(
            f'{key}={json.dumps(value)}' for key, value in debug_env.items()) + '\n')
        self.say('ready', api=api_url, web=web_url, path=self.state / 'settings.json')

    def wait_api(self):
        values = dict(line.split('=', 1) for line in (self.state / 'debug.env').read_text().splitlines())
        url = json.loads(values['CASTOR_API_URL']) + '/ready'
        self.say('wait_api')
        deadline = time.monotonic() + 180
        while time.monotonic() < deadline:
            try:
                with urllib.request.urlopen(url, timeout=2) as response:
                    if response.status == 200:
                        return
            except (urllib.error.URLError, TimeoutError):
                pass
            time.sleep(1)
        raise RuntimeError(self.message('api_timeout', url=url))

    def stop(self):
        if not (self.state / 'settings.json').exists():
            return
        env = self.compose_env(self.settings())
        self.run(self.compose() + ['stop', 'postgres', 'redis', 'minio'], env=env)
        self.say('stopped')


def main():
    dev = DevEnvironment()
    try:
        action = sys.argv[1] if len(sys.argv) == 2 else 'prepare'
        if action == 'prepare':
            dev.prepare()
        elif action == 'wait-api':
            dev.wait_api()
        elif action == 'stop':
            dev.stop()
        else:
            raise RuntimeError(dev.message('usage'))
    except (OSError, ValueError, KeyError, RuntimeError, subprocess.SubprocessError) as error:
        print(dev.message('failed', error=error), file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())

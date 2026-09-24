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
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
# `dev.py run <app>`: working directory and command of each development server.
APPS = {
    'api': ('apps/api', ['go', 'run', './cmd/castor']),
    'web': ('apps/web', ['bun', 'run', 'dev', '--hostname', '127.0.0.1']),
}


class DevEnvironment:
    def __init__(self, root=ROOT):
        self.root = Path(root)
        self.state = self.root / '.local/dev'
        language = os.environ.get('CASTOR_DEV_LANG', os.environ.get('LANG', 'zh')).split('_')[0].split('-')[0]
        self.language = language if language in ('en', 'zh', 'ja', 'ko') else 'zh'
        self.messages = json.loads((ROOT / 'scripts/i18n/dev.json').read_text(encoding='utf-8'))[self.language]

    def message(self, key, **values):
        return self.messages[key].format(**values)

    def say(self, key, **values):
        print(self.message(key, **values), flush=True)

    def run(self, args, **kwargs):
        # Resolve through PATH/PATHEXT so Windows finds bun.cmd-style shims too.
        return subprocess.run([shutil.which(args[0]) or args[0], *args[1:]], check=True, **kwargs)

    def write_private(self, path, value):
        path.write_text(value, encoding='utf-8')
        path.chmod(0o600)

    def settings(self):
        self.state.mkdir(parents=True, exist_ok=True, mode=0o700)
        path = self.state / 'settings.json'
        if path.exists():
            values = json.loads(path.read_text(encoding='utf-8'))
            # Environments prepared before a secret existed get it added once.
            if 'data_key' not in values:
                values['data_key'] = secrets.token_hex(32)
                self.write_private(path, json.dumps(values, indent=2) + '\n')
            return values
        values = {
            'project': 'castor-dev-' + hashlib.sha256(str(self.root).encode()).hexdigest()[:8],
            'api_port': 1234, 'web_port': 3000,
            'jwt_key': secrets.token_hex(32), 'rsa_secret': secrets.token_hex(16),
            'data_key': secrets.token_hex(32),
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
            # Probe the way Node and Go listen: on POSIX a just-stopped server's TIME_WAIT
            # connections would otherwise make its port look taken. (On Windows the option
            # means "share a port in use", and TIME_WAIT never blocks a bind there.)
            if os.name != 'nt':
                sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
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
            'RUSTFS_API_PORT': '0', 'RUSTFS_CONSOLE_PORT': '0',
        }
        self.write_env(self.state / 'compose.env', values)
        return dict(os.environ, **values)

    def write_env(self, path, values):
        # One double-quoted value per line: read by docker compose --env-file, VS Code envFile
        # and `dev.py run`. All values are generated here (hex secrets, ports, paths); paths use
        # forward slashes, which every consumer accepts on Windows without escaping.
        self.write_private(path, ''.join(f'{key}={json.dumps(value)}\n' for key, value in values.items()))

    def read_env(self, path):
        values = {}
        for line in path.read_text(encoding='utf-8').splitlines():
            key, _, value = line.partition('=')
            values[key] = json.loads(value)
        return values

    def published_port(self, service, port, env):
        address = self.run(self.compose() + ['port', service, str(port)],
                           env=env, capture_output=True, text=True).stdout.strip()
        return int(address.rsplit(':', 1)[1])

    @staticmethod
    def add_template_fields(text, template, keys):
        """Copy fields the local config lacks (it predates them) from the template, at the end of their section."""
        def fields(source):
            section, found = '', {}
            for index, line in enumerate(source.splitlines()):
                header = re.match(r'^\[([^]]+)\]', line)
                if header:
                    section = header.group(1)
                field = re.match(r'^(\w+)\s*=', line)
                if field:
                    found[(section, field.group(1))] = (index, line)
            return found
        present, available = fields(text), fields(template)
        lines = text.splitlines()
        for key in keys:
            if key in present or key not in available:
                continue
            section = [index for (sec, _), (index, _) in fields('\n'.join(lines)).items() if sec == key[0]]
            line = available[key][1]
            if section:
                lines.insert(max(section) + 1, line)
            else:
                lines += ['', f'[{key[0]}]', line]
        return '\n'.join(lines) + '\n'

    @staticmethod
    def example_mail_host(text):
        """Whether Mail.Host is a documentation domain (RFC 2606), as the API's config check sees it."""
        import tomllib

        host = str(tomllib.loads(text).get('Mail', {}).get('Host', '')).strip().lower().rstrip('.')
        return any(host == reserved or host.endswith('.' + reserved)
                   for reserved in ('example.com', 'example.net', 'example.org', 'example', 'invalid'))

    @staticmethod
    def configure_toml(text, values):
        import tomllib

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

    # Claude Code 的 Browser pane 从 .claude/launch.json 读取启动配置。端口由本脚本
    # 选定（1234/3000 被占用时会改选），所以这个文件必须跟着一起生成：手写的副本迟早
    # 会和实际端口对不上，而 preview 打开错误端口时不会报错，只会显示一张空白页。
    # 它因此是生成物（已在 .gitignore 中），通过 `dev.py run` 读取 debug.env，不含任何凭据；
    # 不经过 shell，Windows / macOS / Linux 上是同一份配置。
    def write_claude_launch(self, settings):
        launch = {
            'version': '0.0.1',
            'configurations': [{
                'name': f'castor-{app}',
                'runtimeExecutable': sys.executable,
                'runtimeArgs': [str(self.root / 'scripts/dev.py'), 'run', app],
                'port': settings[f'{app}_port'],
            } for app in APPS],
        }
        path = self.root / '.claude/launch.json'
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(launch, indent=2) + '\n', encoding='utf-8')

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
                                  'postgres', 'redis', 'rustfs'], env=env)
        self.run(self.compose() + ['run', '--rm', '--no-deps', 'rustfs-init'], env=env)
        ports = {service: self.published_port(service, port, env)
                 for service, port in [('postgres', 5432), ('redis', 6379), ('rustfs', 9000)]}
        api_url = f'http://127.0.0.1:{settings["api_port"]}'
        web_url = f'http://127.0.0.1:{settings["web_port"]}'
        config_path = self.state / 'api.config.toml'
        source = config_path if config_path.exists() else self.root / 'deploy/config/api.config.example.toml'
        values = {
            ('General', 'Address'): f'127.0.0.1:{settings["api_port"]}',
            ('General', 'Development'): True,
            ('General', 'JwtKey'): settings['jwt_key'],
            ('General', 'PublicURL'): web_url,
            ('Security', 'RSAPrivateKeySecret'): settings['rsa_secret'],
            ('Security', 'DataEncryptionKey'): settings['data_key'],
            ('CORS', 'AllowOrigins'): [web_url, f'http://localhost:{settings["web_port"]}'],
            ('Postgres', 'Host'): '127.0.0.1', ('Postgres', 'Port'): ports['postgres'],
            ('Postgres', 'Username'): 'castor', ('Postgres', 'DbName'): 'castor',
            ('Postgres', 'Password'): settings['postgres_password'],
            ('Redis', 'Address'): f'127.0.0.1:{ports["redis"]}',
            ('Redis', 'Password'): settings['redis_password'],
            ('S3', 'Endpoint'): f'127.0.0.1:{ports["rustfs"]}',
            ('S3', 'AccessKey'): 'castor', ('S3', 'Secret'): settings['s3_secret'],
            ('S3', 'Bucket'): 'castor',
        }
        # Configs generated from older templates carry the example SMTP host, which the API rejects.
        if self.example_mail_host(source.read_text(encoding='utf-8')):
            values[('Mail', 'Host')] = ''
        template = (self.root / 'deploy/config/api.config.example.toml').read_text(encoding='utf-8')
        text = self.add_template_fields(source.read_text(encoding='utf-8'), template, values)
        self.write_private(config_path, self.configure_toml(text, values))
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
            'CASTOR_CONFIG_FILE': config_path.as_posix(), 'CASTOR_API_URL': api_url,
            'PORT': str(settings['web_port']), 'NODE_ENV': 'development',
            'NEXT_PUBLIC_API_URL': '', 'NEXT_PUBLIC_SENTRY_DISABLED': 'true',
            'BUILD_STANDALONE': 'false',
        }
        # Explicitly supply API overrides so inherited terminal variables cannot change the target DB.
        debug_env.update({
            'CASTOR_JWT_KEY': settings['jwt_key'], 'CASTOR_RSA_SECRET': settings['rsa_secret'],
            'CASTOR_DATA_KEY': settings['data_key'], 'CASTOR_PUBLIC_URL': web_url,
            'CASTOR_DB_HOST': '127.0.0.1', 'CASTOR_DB_PORT': str(ports['postgres']),
            'CASTOR_DB_USER': 'castor', 'CASTOR_DB_NAME': 'castor',
            'CASTOR_DB_PASSWORD': settings['postgres_password'],
            'CASTOR_REDIS_ADDR': f'127.0.0.1:{ports["redis"]}',
            'CASTOR_REDIS_PASSWORD': settings['redis_password'],
            'CASTOR_S3_ENDPOINT': f'127.0.0.1:{ports["rustfs"]}',
            'CASTOR_S3_ACCESS_KEY': 'castor', 'CASTOR_S3_SECRET_KEY': settings['s3_secret'],
            'CASTOR_S3_BUCKET': 'castor', 'CASTOR_CORS_ORIGINS': web_url,
        })
        self.write_env(self.state / 'debug.env', debug_env)
        self.write_claude_launch(settings)
        self.say('ready', api=api_url, web=web_url, path=self.state / 'settings.json')

    def start(self, app):
        """Run a development server with the environment from debug.env (no shell needed)."""
        path = self.state / 'debug.env'
        if not path.exists():
            raise RuntimeError(self.message('not_prepared'))
        # Inherited CASTOR_* values must not redirect the server to another database.
        env = {key: value for key, value in os.environ.items() if not key.startswith('CASTOR_')}
        env.update(self.read_env(path))
        workdir, command = APPS[app]
        cwd = self.root / workdir
        command = [shutil.which(command[0]) or command[0], *command[1:]]
        if os.name == 'posix':
            # Become the server, so stopping this process stops the server itself.
            os.chdir(cwd)
            os.execve(command[0], command, env)
        return subprocess.run(command, cwd=cwd, env=env).returncode

    def wait_api(self):
        url = f'http://127.0.0.1:{self.settings()["api_port"]}/ready'
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
        self.run(self.compose() + ['stop', 'postgres', 'redis', 'rustfs'], env=env)
        self.say('stopped')


def main():
    for stream in (sys.stdout, sys.stderr):
        # Consoles on a legacy code page must not crash on translated messages.
        stream.reconfigure(errors='replace')
    dev = DevEnvironment()
    try:
        if sys.version_info < (3, 11):
            raise RuntimeError(dev.message('python_version',
                                           version='.'.join(map(str, sys.version_info[:3])),
                                           executable=sys.executable))
        args = sys.argv[1:] or ['prepare']
        if args == ['prepare']:
            dev.prepare()
        elif args == ['wait-api']:
            dev.wait_api()
        elif args == ['stop']:
            dev.stop()
        elif len(args) == 2 and args[0] == 'run' and args[1] in APPS:
            return dev.start(args[1])
        else:
            raise RuntimeError(dev.message('usage'))
    except (OSError, ValueError, KeyError, RuntimeError, subprocess.SubprocessError) as error:
        print(dev.message('failed', error=error), file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())

"""Check release ordering and failure gates without contacting servers or clusters.

The scripts run in-process against a copy of deploy/ with every external tool replaced by a
recorder, so the same tests pass on Windows, macOS and Linux without Docker or kubectl.
"""
import ast
import contextlib
import gzip
import io
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'scripts'))
import _tooling as tooling  # noqa: E402

SCRIPTS = ('backup', 'build-images', 'compose-instance', 'deploy-compose', 'deploy-k8s')


class Stdin(io.BytesIO):
    def close(self):
        pass  # keep what the script streamed for the assertions


class FakeProcess:
    def __init__(self, test, args, kwargs):
        self.args = args
        output, self._code = test.respond(args)
        self.stdin = Stdin()
        test.stdins.append(self.stdin)
        target = kwargs.get('stdout')
        self.stdout = io.BytesIO(output) if target == subprocess.PIPE else None
        if target not in (None, subprocess.PIPE, subprocess.DEVNULL):
            target.write(output)
        if kwargs.get('stdin') not in (None, subprocess.PIPE):
            self.stdin.write(kwargs['stdin'].read())
        self.returncode = None

    def communicate(self, input=None):
        if input is not None:
            self.stdin.write(input)
        self.returncode = self._code
        return (self.stdout.read() if self.stdout else None), None

    def wait(self):
        self.returncode = self._code
        return self._code


class Result:
    def __init__(self, returncode, stdout, stderr):
        self.returncode, self.stdout, self.stderr = returncode, stdout, stderr


class DeploymentTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='castor-release-test-')
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        shutil.copytree(ROOT / 'deploy', self.root / 'deploy', ignore=shutil.ignore_patterns(
            'backups', 'artifacts', 'instances', '.env', '.env.shared', 'api.config.toml',
            'secrets.env', 'bootstrap.env'))
        compose = self.root / 'deploy/compose'
        (compose / '.env').write_text('CASTOR_JWT_KEY=' + 'c' * 32 + '\n', encoding='utf-8')
        (compose / 'config/api.config.toml').write_text('[General]\n', encoding='utf-8')
        for stage in ('staging', 'production'):
            overlay = self.root / 'deploy/k8s/overlays' / stage
            for name in ('secrets', 'bootstrap'):
                (overlay / (name + '.env')).write_text(
                    (overlay / (name + '.env.example')).read_text(encoding='utf-8').replace('=\n', '=' + 'a' * 32 + '\n'),
                    encoding='utf-8')
        self.log = []
        self.stdins = []
        self.fail = None
        self.fail_target = None
        self.running = ''
        environment = patch.dict(os.environ)
        environment.start()
        self.addCleanup(environment.stop)
        for key in ('SHARED_ENV_FILE', 'BACKUP_ROOT', 'PLATFORM'):
            os.environ.pop(key, None)

    def respond(self, args):
        """What each fake tool prints and returns."""
        if 'kustomize' in args:
            return b'apiVersion: v1\nkind: Namespace\nmetadata:\n  name: castor-staging\n', 0
        if self.fail == 'init' and 'run' in args and 'init-db' in args:
            return b'', 1
        if self.fail_target and any(self.fail_target in part for part in args) and (
                ('run' in args and 'init-db' in args) or 'wait' in args):
            return b'', 1
        if self.fail == 'job' and 'wait' in args:
            return b'', 1
        if self.fail == 'volume' and args[:2] == ['docker', 'run']:
            return b'', 1
        if 'ps' in args and '-aq' in args:
            return b'container-' + args[-1].encode(), 0
        if 'ps' in args and '--status' in args:
            return self.running.encode(), 0
        if args[:2] == ['docker', 'inspect']:
            return b'sha256:image-' + args[-1].encode(), 0
        if args[:2] == ['docker', 'save']:
            return b'image layers', 0
        return b'', 0

    def fake(self, args, **kwargs):
        self.log.append(list(args))
        return FakeProcess(self, list(args), kwargs)

    def run_script(self, script, *args):
        module = tooling.load_script(script)
        stdout, stderr = io.StringIO(), io.StringIO()
        with patch.object(tooling, 'ROOT', self.root), patch.object(tooling, 'process', self.fake), \
                contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
            code = tooling.cli(module.main, [str(arg) for arg in args])
        return Result(code, stdout.getvalue(), stderr.getvalue())

    def calls(self):
        return self.log

    def test_scripts_run_on_the_python_of_long_term_support_servers(self):
        # Deployment scripts run on Docker hosts, where Python 3.9 is still common.
        for script in SCRIPTS + ('_tooling',):
            with self.subTest(script=script):
                source = (ROOT / 'scripts' / f'{script}.py').read_text(encoding='utf-8')
                ast.parse(source, feature_version=(3, 9))

    def test_only_git_hooks_remain_shell_scripts(self):
        self.assertEqual(sorted(p.name for p in (ROOT / 'scripts').rglob('*.sh')), [])

    def test_compose_init_failure_does_not_update_app(self):
        self.fail = 'init'
        self.assertNotEqual(self.run_script('deploy-compose', 'up', 'full').returncode, 0)
        self.assertTrue(any('init-db' in call for call in self.calls()))
        self.assertFalse(any('up' in call and 'castor-api' in call for call in self.calls()))

    def test_compose_reruns_init_on_each_release(self):
        for _ in range(2):
            self.assertEqual(self.run_script('deploy-compose', 'up', 'external').returncode, 0)
        calls = self.calls()
        init = [i for i, c in enumerate(calls) if 'run' in c and 'init-db' in c]
        app = [i for i, c in enumerate(calls) if 'up' in c and 'castor-api' in c]
        self.assertEqual(len(init), 2)
        self.assertEqual(len(app), 2)
        self.assertTrue(init[0] < app[0] < init[1] < app[1])

    def test_k8s_failed_job_preserves_deployments_and_releases_lock(self):
        self.fail = 'job'
        self.assertNotEqual(self.run_script('deploy-k8s', 'apply', 'staging', 'test-context').returncode, 0)
        self.assertFalse(any('castor.io/phase=application' in c for c in self.calls()))
        self.assertTrue(any('delete' in c and 'castor-release-lock' in c for c in self.calls()))

    def test_k8s_waits_for_init_before_app_and_uses_explicit_context(self):
        result = self.run_script('deploy-k8s', 'apply', 'staging', 'test-context')
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = self.calls()
        wait = next(i for i, c in enumerate(calls) if 'wait' in c)
        app = next(i for i, c in enumerate(calls) if 'castor.io/phase=application' in c)
        self.assertLess(wait, app)
        for c in calls:
            if 'kustomize' not in c:
                self.assertEqual(c[1:3], ['--context', 'test-context'])

    def test_k8s_requires_context_before_contacting_cluster(self):
        self.assertNotEqual(self.run_script('deploy-k8s', 'apply', 'staging').returncode, 0)
        self.assertEqual(self.calls(), [])

    def test_k8s_render_to_file_is_private(self):
        output = self.root / 'rendered.yaml'
        self.assertEqual(self.run_script('deploy-k8s', 'render', 'staging', output).returncode, 0)
        self.assertIn('kind: Namespace', output.read_text(encoding='utf-8'))
        if os.name == 'posix':
            self.assertEqual(output.stat().st_mode & 0o777, 0o600)

    def test_rollback_never_runs_old_initialization(self):
        for script, args in (
            ('deploy-compose', ('rollback', 'full')),
            ('deploy-k8s', ('rollback', 'staging', 'test-context')),
        ):
            result = self.run_script(script, *args)
            self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(any('run' in c and 'init-db' in c for c in self.calls()))
        self.assertFalse(any('create' in c and 'job' in c for c in self.calls()))

    def test_k8s_empty_secret_fails_before_cluster_mutation(self):
        (self.root / 'deploy/k8s/overlays/staging/secrets.env').write_text('CASTOR_JWT_KEY=\n', encoding='utf-8')
        self.assertNotEqual(self.run_script('deploy-k8s', 'apply', 'staging', 'test-context').returncode, 0)
        self.assertFalse(any('apply' in c or 'create' in c for c in self.calls()))

    def test_remote_runs_compose_over_ssh_with_posix_paths(self):
        result = self.run_script('deploy-compose', 'remote', 'registry', 'deploy@example.com', '/opt/castor', 'full')
        self.assertEqual(result.returncode, 0, result.stderr)
        remote = [c[2] for c in self.calls() if c[0] == 'ssh']
        self.assertTrue(all(c[1] == 'deploy@example.com' for c in self.calls() if c[0] == 'ssh'))
        pull = next(i for i, c in enumerate(remote) if ' pull' in c)
        init = next(i for i, c in enumerate(remote) if 'init-db' in c)
        app = next(i for i, c in enumerate(remote) if 'castor-api' in c)
        self.assertTrue(pull < init < app)
        # Remote hosts are Linux: paths must stay POSIX even when deploying from Windows.
        self.assertIn('-f /opt/castor/deploy/compose/compose.infra.yaml', remote[app])
        # Configuration and secrets travel inside the tar stream, not on the command line.
        shipped = next(s.getvalue() for s, c in zip(self.stdins, self.calls()) if 'tar -xf' in ' '.join(c))
        with tarfile.open(fileobj=io.BytesIO(shipped)) as bundle:
            self.assertIn('deploy/compose/.env', bundle.getnames())
        self.assertFalse(any('c' * 32 in ' '.join(c) for c in self.calls()))

    def test_remote_tar_mode_streams_the_archive_to_docker_load(self):
        archive = self.root / 'images.tar.gz'
        archive.write_bytes(gzip.compress(b'images'))
        result = self.run_script('deploy-compose', 'remote', 'tar', 'deploy@example.com', '/opt/castor',
                                 'external', archive)
        self.assertEqual(result.returncode, 0, result.stderr)
        load = next(i for i, c in enumerate(self.calls()) if c[-1] == 'docker load')
        self.assertEqual(self.stdins[load].getvalue(), archive.read_bytes())
        self.assertFalse(any(' pull' in c[-1] for c in self.calls() if c[0] == 'ssh'))

    def test_remote_rejects_unsafe_destination_before_connecting(self):
        for dest in ('relative/path', "/opt/castor'; rm -rf /"):
            with self.subTest(dest=dest):
                result = self.run_script('deploy-compose', 'remote', 'registry', 'deploy@example.com', dest, 'full')
                self.assertEqual(result.returncode, 2)
        self.assertEqual(self.calls(), [])

    def test_build_images_tar_archives_both_images(self):
        archive = self.root / 'castor.tar.gz'
        result = self.run_script('build-images', 'tar', 'registry.example.com/castor/', 'v1', archive)
        self.assertEqual(result.returncode, 0, result.stderr)
        builds = [c for c in self.calls() if c[:3] == ['docker', 'buildx', 'build']]
        self.assertEqual([c[c.index('-t') + 1] for c in builds],
                         ['registry.example.com/castor/castor-api:v1', 'registry.example.com/castor/castor-web:v1'])
        self.assertEqual(gzip.decompress(archive.read_bytes()), b'image layers')

    def test_build_images_forwards_mirror_build_args_to_the_api_only(self):
        with patch.dict(os.environ, {'GOPROXY': 'https://goproxy.cn,direct', 'UBUNTU_MIRROR': 'mirrors.example.com'}):
            os.environ.pop('UBUNTU_SECURITY_MIRROR', None)
            result = self.run_script('build-images', 'load', 'castor', 'v1')
        self.assertEqual(result.returncode, 0, result.stderr)
        api, web = [c for c in self.calls() if c[:3] == ['docker', 'buildx', 'build']]
        self.assertIn('GOPROXY=https://goproxy.cn,direct', api)
        self.assertIn('UBUNTU_MIRROR=mirrors.example.com', api)
        self.assertFalse(any(arg.startswith('UBUNTU_SECURITY_MIRROR=') for arg in api))
        self.assertNotIn('--build-arg', web)

    def test_build_images_tar_requires_archive_before_building(self):
        self.assertEqual(self.run_script('build-images', 'tar', 'castor', 'v1').returncode, 2)
        self.assertEqual(self.calls(), [])

    def shared_and_instance(self, instance='tenant-a', port='8081'):
        shared = self.root / 'deploy/compose/.env.shared'
        shared.write_text((self.root / 'deploy/compose/.env.shared.example').read_text(encoding='utf-8').replace('=\n', '=' + 'b' * 32 + '\n'), encoding='utf-8')
        result = self.run_script('compose-instance', 'new', instance, port)
        self.assertEqual(result.returncode, 0, result.stderr)
        return Path(result.stdout.strip())

    def test_instance_env_is_private_and_unique(self):
        env_file = self.shared_and_instance()
        if os.name == 'posix':
            self.assertEqual(env_file.stat().st_mode & 0o777, 0o600)
        self.assertNotIn(b'\r', env_file.read_bytes())
        values = dict(line.split('=', 1) for line in env_file.read_text(encoding='utf-8').splitlines() if '=' in line and not line.startswith('#'))
        self.assertEqual(values['CASTOR_INSTANCE_ID'], 'tenant-a')
        self.assertEqual(values['POSTGRES_DB'], 'castor_tenant_a')
        self.assertEqual(values['S3_BUCKET'], 'castor-tenant-a')
        self.assertEqual(values['REDIS_PASSWORD'], 'b' * 32)  # shared Redis, own namespace
        other = self.shared_and_instance('tenant-b', '8082')
        other_values = dict(line.split('=', 1) for line in other.read_text(encoding='utf-8').splitlines() if '=' in line and not line.startswith('#'))
        self.assertEqual(len(values['CASTOR_DATA_KEY']), 64)
        for key in ('CASTOR_JWT_KEY', 'CASTOR_RSA_SECRET', 'CASTOR_DATA_KEY', 'POSTGRES_PASSWORD', 'S3_SECRET_KEY'):
            self.assertNotEqual(values[key], other_values[key], key)
        # Never overwrite an existing instance, never accept an id that breaks the Redis prefix.
        self.assertNotEqual(self.run_script('compose-instance', 'new', 'tenant-a', '8083').returncode, 0)
        self.assertNotEqual(self.run_script('compose-instance', 'new', 'Tenant:A', '8083').returncode, 0)

    def test_compose_instance_provisions_before_init_without_secrets_in_argv(self):
        env_file = self.shared_and_instance()
        result = self.run_script('deploy-compose', 'up', 'instance', env_file)
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = self.calls()
        provision = [i for i, c in enumerate(calls) if 'exec' in c]
        bucket = next(i for i, c in enumerate(calls) if c[:3] == ['docker', 'run', '--rm'] and 'rustfs/rc' in ' '.join(c))
        init = next(i for i, c in enumerate(calls) if 'run' in c and 'init-db' in c)
        app = next(i for i, c in enumerate(calls) if 'up' in c and 'castor-api' in c)
        self.assertEqual(len(provision), 1)  # database; the bucket is provisioned by an rc container
        self.assertTrue(max(provision + [bucket]) < init < app)
        for c in calls[init], calls[app]:
            self.assertIn(str(self.root / 'deploy/compose/compose.instance.yaml'), c)
        secrets = [line.split('=', 1)[1] for line in env_file.read_text(encoding='utf-8').splitlines()
                   if line.split('=', 1)[0] in ('POSTGRES_PASSWORD', 'S3_SECRET_KEY')]
        for c in calls:
            for secret in secrets:
                self.assertNotIn(secret, ' '.join(c))
        # They reach psql and rc on stdin instead.
        for index, secret in zip(provision + [bucket], secrets):
            self.assertIn(secret.encode(), self.stdins[index].getvalue())

    def test_compose_shared_starts_infrastructure_only(self):
        self.shared_and_instance()
        result = self.run_script('deploy-compose', 'up', 'shared')
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = self.calls()
        self.assertTrue(any('up' in c and 'postgres' in c for c in calls))
        for c in calls:
            self.assertNotIn('init-db', c)
            self.assertNotIn('castor-api', c)
            self.assertNotIn('rustfs-init', c)  # buckets are created per instance
        self.assertNotEqual(self.run_script('deploy-compose', 'rollback', 'shared').returncode, 0)

    def test_instance_restore_requires_confirmation(self):
        env_file = self.shared_and_instance()
        snapshot = self.root / 'deploy/backups/instances/tenant-a/snapshot-20260921_120000'
        (snapshot / 'bucket').mkdir(parents=True)
        (snapshot / 'database.dump').write_text('dump', encoding='utf-8')
        (snapshot / 'complete').touch()
        for extra in ((), ('--confirm', 'tenant-b/snapshot-20260921_120000')):
            result = self.run_script('compose-instance', 'restore', env_file, snapshot.name, *extra)
            self.assertEqual(result.returncode, 2)
            self.assertIn('--confirm tenant-a/snapshot-20260921_120000', result.stderr)
        self.assertEqual(self.calls(), [])

    def test_instance_backup_restarts_api_and_stores_dump(self):
        env_file = self.shared_and_instance()
        self.running = 'api-container'
        result = self.run_script('compose-instance', 'backup', env_file)
        self.assertEqual(result.returncode, 0, result.stderr)
        snapshot = Path(result.stdout.strip())
        self.assertTrue((snapshot / 'complete').is_file())
        self.assertEqual((snapshot / 'instance.env').read_text(encoding='utf-8'), env_file.read_text(encoding='utf-8'))
        calls = self.calls()
        stop = next(i for i, c in enumerate(calls) if 'stop' in c)
        dump = next(i for i, c in enumerate(calls) if any('pg_dump' in part for part in c))
        start = next(i for i, c in enumerate(calls) if 'start' in c)
        self.assertTrue(stop < dump < start)

    def test_instance_backup_uses_pinned_rc_image(self):
        infra = (ROOT / 'deploy/compose/compose.infra.yaml').read_text(encoding='utf-8')
        self.assertIn('image: ' + tooling.load_script('compose-instance').RC_IMAGE, infra)

    @staticmethod
    def volume_archives(snapshot):
        """The helper container writes the volume archives; stand in for it."""
        for service in ('postgres', 'redis', 'rustfs'):
            with tarfile.open(snapshot / f'{service}.tar', 'w') as archive:
                archive.addfile(tarfile.TarInfo('data'), io.BytesIO())

    def test_backup_create_then_restore_with_confirmation(self):
        self.running = 'postgres\nredis\nrustfs\ncastor-api\ngateway\n'
        created = self.run_script('backup', 'create', 'full')
        self.assertEqual(created.returncode, 0, created.stderr)
        snapshot = Path(created.stdout.strip())
        with tarfile.open(snapshot / 'config.tar') as config:
            self.assertIn('.env', config.getnames())
        calls = self.calls()
        stops = [c for c in calls if 'stop' in c]
        self.assertEqual(stops[0][-2:], ['castor-api', 'gateway'])  # application drains first
        self.assertTrue(any('start' in c for c in calls))

        self.volume_archives(snapshot)
        self.log.clear()
        self.assertEqual(self.run_script('backup', 'restore', 'full', snapshot.name).returncode, 2)
        self.assertEqual(self.calls(), [])
        restored = self.run_script('backup', 'restore', 'full', snapshot.name, '--confirm', snapshot.name)
        self.assertEqual(restored.returncode, 0, restored.stderr)

    def test_failed_restore_leaves_services_stopped(self):
        self.running = 'postgres\nredis\nrustfs\ncastor-api\n'
        snapshot = Path(self.run_script('backup', 'create', 'full').stdout.strip())
        self.volume_archives(snapshot)
        self.log.clear()
        self.fail = 'volume'
        result = self.run_script('backup', 'restore', 'full', snapshot.name, '--confirm', snapshot.name)
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(any('stop' in c for c in self.calls()))
        self.assertFalse(any('start' in c for c in self.calls()))


    # ---- one instance per tenant: batch operations ----

    def test_instance_status_lists_every_instance(self):
        self.shared_and_instance('tenant-b', '8082')
        self.shared_and_instance('tenant-a', '8081')
        self.running = 'api-container'
        result = self.run_script('compose-instance', 'status')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.splitlines(), ['tenant-a\t8081\trunning', 'tenant-b\t8082\trunning'])

    def test_all_upgrades_instances_in_order_and_stops_at_the_first_failure(self):
        for name, port in (('tenant-a', '8081'), ('tenant-b', '8082'), ('tenant-c', '8083')):
            self.shared_and_instance(name, port)
        self.fail_target = 'tenant-b.env'
        result = self.run_script('compose-instance', 'all', 'up')
        self.assertEqual(result.returncode, 1)
        self.assertIn('ok\ttenant-a', result.stdout)
        self.assertIn('FAILED\ttenant-b', result.stdout)
        self.assertIn('skipped\ttenant-c', result.stdout)
        inits = [c for c in self.calls() if 'run' in c and 'init-db' in c]
        self.assertEqual([Path(c[c.index('--env-file') + 1]).stem for c in inits], ['tenant-a', 'tenant-b'])
        # A failed init never updates that instance's application.
        started = [Path(c[c.index('--env-file') + 1]).stem for c in self.calls() if 'up' in c and 'castor-api' in c]
        self.assertEqual(started, ['tenant-a'])

    def test_all_keep_going_continues_but_still_fails(self):
        for name, port in (('tenant-a', '8081'), ('tenant-b', '8082'), ('tenant-c', '8083')):
            self.shared_and_instance(name, port)
        self.fail_target = 'tenant-b.env'
        result = self.run_script('compose-instance', 'all', 'up', '--keep-going')
        self.assertEqual(result.returncode, 1)
        self.assertIn('ok\ttenant-c', result.stdout)
        self.assertNotIn('skipped', result.stdout)

    def test_all_backup_snapshots_every_instance(self):
        self.shared_and_instance('tenant-a', '8081')
        self.shared_and_instance('tenant-b', '8082')
        result = self.run_script('compose-instance', 'all', 'backup')
        self.assertEqual(result.returncode, 0, result.stderr)
        for name in ('tenant-a', 'tenant-b'):
            snapshots = tooling.snapshots(self.root / 'deploy/backups/instances' / name)
            self.assertEqual(len(snapshots), 1, name)
            self.assertTrue((snapshots[0] / 'complete').is_file())

    def test_all_rejects_unknown_actions_and_empty_fleets(self):
        self.assertEqual(self.run_script('compose-instance', 'all', 'up').returncode, 2)  # no instances yet
        self.shared_and_instance()
        for args in (('all', 'restore'), ('all', 'up', '--force')):
            self.assertEqual(self.run_script('compose-instance', *args).returncode, 2, args)
        self.assertEqual(self.calls(), [])

    def k8s_instance(self, instance='tenant-a', domain='tenant-a.example.com', fill=True):
        result = self.run_script('deploy-k8s', 'new', instance, domain)
        self.assertEqual(result.returncode, 0, result.stderr)
        target = Path(result.stdout.strip())
        if fill:
            for name in ('secrets.env', 'bootstrap.env'):
                path = target / name
                path.write_text(path.read_text(encoding='utf-8').replace('=\n', '=' + 'd' * 32 + '\n'), encoding='utf-8')
        return target

    def test_k8s_new_instance_overlay(self):
        target = self.k8s_instance(fill=False)
        self.assertEqual(target, self.root / 'deploy/k8s/instances/tenant-a')
        kustomization = (target / 'kustomization.yaml').read_text(encoding='utf-8')
        self.assertIn('namespace: castor-tenant-a\n', kustomization)
        self.assertEqual(kustomization.count('value: tenant-a.example.com\n'), 2)
        self.assertIn('value: tenant-a\n', kustomization)  # CASTOR_INSTANCE_ID
        self.assertNotIn('castor-production', kustomization)
        self.assertIn('name: castor-tenant-a\n', (target / 'namespace.yaml').read_text(encoding='utf-8'))
        # Everything the production overlay references comes along.
        for name in ('autoscaling.yaml', 'pod-anti-affinity.yaml', 'pod-disruption-budgets.yaml', 'api.config.toml'):
            self.assertTrue((target / name).is_file(), name)
        secrets_env = dict(line.split('=', 1) for line in (target / 'secrets.env').read_text(encoding='utf-8').splitlines())
        self.assertEqual(len(secrets_env['CASTOR_JWT_KEY']), 64)
        self.assertEqual(len(secrets_env['CASTOR_RSA_SECRET']), 32)
        self.assertEqual(len(secrets_env['CASTOR_DATA_KEY']), 64)
        self.assertIn('PublicURL = "https://tenant-a.example.com"', (target / 'api.config.toml').read_text(encoding='utf-8'))
        self.assertEqual(secrets_env['CASTOR_DB_PASSWORD'], '')  # external credential, filled in by the operator
        if os.name == 'posix':
            for name in ('secrets.env', 'bootstrap.env'):
                self.assertEqual((target / name).stat().st_mode & 0o777, 0o600, name)
        other = self.k8s_instance('tenant-b', 'tenant-b.example.com', fill=False)
        self.assertNotEqual((target / 'secrets.env').read_text(encoding='utf-8'), (other / 'secrets.env').read_text(encoding='utf-8'))
        for args in (('new', 'tenant-a', 'x.example.com'), ('new', 'staging', 'x.example.com'),
                     ('new', 'Tenant', 'x.example.com'), ('new', 'tenant-c', 'not a domain')):
            self.assertEqual(self.run_script('deploy-k8s', *args).returncode, 2, args)

    def test_k8s_instance_needs_external_secrets_before_touching_the_cluster(self):
        self.k8s_instance(fill=False)
        result = self.run_script('deploy-k8s', 'apply', 'tenant-a', 'ctx')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(any('apply' in c for c in self.calls()))

    def test_k8s_instance_deploys_into_its_own_namespace(self):
        target = self.k8s_instance()
        result = self.run_script('deploy-k8s', 'apply', 'tenant-a', 'ctx')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(any('kustomize' in c and str(target) in c for c in self.calls()))
        namespaced = [c for c in self.calls() if '-n' in c]
        self.assertTrue(namespaced)
        self.assertTrue(all(c[c.index('-n') + 1] == 'castor-tenant-a' for c in namespaced))
        self.assertEqual(self.run_script('deploy-k8s', 'apply', 'tenant-z', 'ctx').returncode, 2)

    def test_k8s_apply_all_in_order_and_stops_at_the_first_failure(self):
        for name in ('tenant-a', 'tenant-b', 'tenant-c'):
            self.k8s_instance(name, f'{name}.example.com')
        self.fail_target = 'castor-tenant-b'
        result = self.run_script('deploy-k8s', 'apply-all', 'ctx')
        self.assertEqual(result.returncode, 1)
        self.assertEqual([line for line in result.stdout.splitlines() if '\t' in line],
                         ['ok\ttenant-a', 'FAILED\ttenant-b', 'skipped\ttenant-c'])
        # The failing instance's init job gates its application; the next instance is untouched.
        applied = [c[c.index('-n') + 1] for c in self.calls() if 'rollout' in c]
        self.assertEqual(set(applied), {'castor-tenant-a'})
        result = self.run_script('deploy-k8s', 'apply-all', 'ctx', '--keep-going')
        self.assertIn('ok\ttenant-c', result.stdout)
        self.assertEqual(self.run_script('deploy-k8s', 'apply-all').returncode, 2)  # explicit context


if __name__ == '__main__':
    unittest.main()

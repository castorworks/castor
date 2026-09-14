"""Check release ordering and failure gates without contacting servers or clusters."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]

MOCK = '''#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
args = sys.argv[1:]
with open(os.environ['CALL_LOG'], 'a') as f:
    f.write(json.dumps([Path(sys.argv[0]).name] + args) + '\\n')
if args and args[0] == 'kustomize':
    print('apiVersion: v1\\nkind: Namespace\\nmetadata:\\n  name: castor-staging')
if os.environ.get('FAIL_STAGE') == 'init' and 'run' in args and 'init-db' in args:
    sys.exit(1)
if os.environ.get('FAIL_STAGE') == 'job' and 'wait' in args:
    sys.exit(1)
'''


class DeploymentTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='castor-release-test-')
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        shutil.copytree(ROOT / 'scripts', self.root / 'scripts')
        shutil.copytree(ROOT / 'deploy', self.root / 'deploy', ignore=shutil.ignore_patterns('backups', 'artifacts', '.env', 'api.config.toml', 'secrets.env', 'bootstrap.env'))
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        for tool in ('docker', 'kubectl'):
            path = self.bin / tool
            path.write_text(MOCK)
            path.chmod(0o755)
        self.log = self.root / 'calls.jsonl'
        self.env = dict(os.environ, PATH=str(self.bin) + ':' + os.environ['PATH'], CALL_LOG=str(self.log))
        for stage in ('staging', 'production'):
            overlay = self.root / 'deploy/k8s/overlays' / stage
            for name in ('secrets', 'bootstrap'):
                (overlay / (name + '.env')).write_text((overlay / (name + '.env.example')).read_text().replace('=\n', '=' + 'a' * 32 + '\n'))

    def run_script(self, script, *args):
        return subprocess.run(['bash', str(self.root / 'scripts' / script), *args], env=self.env, capture_output=True, text=True)

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()] if self.log.exists() else []

    def test_compose_init_failure_does_not_update_app(self):
        self.env['FAIL_STAGE'] = 'init'
        self.assertNotEqual(self.run_script('deploy-compose.sh', 'up', 'full').returncode, 0)
        self.assertTrue(any('init-db' in call for call in self.calls()))
        self.assertFalse(any('up' in call and 'castor-api' in call for call in self.calls()))

    def test_compose_reruns_init_on_each_release(self):
        for _ in range(2):
            self.assertEqual(self.run_script('deploy-compose.sh', 'up', 'external').returncode, 0)
        calls = self.calls()
        init = [i for i, c in enumerate(calls) if 'run' in c and 'init-db' in c]
        app = [i for i, c in enumerate(calls) if 'up' in c and 'castor-api' in c]
        self.assertEqual(len(init), 2)
        self.assertEqual(len(app), 2)
        self.assertTrue(init[0] < app[0] < init[1] < app[1])

    def test_k8s_failed_job_preserves_deployments_and_releases_lock(self):
        self.env['FAIL_STAGE'] = 'job'
        self.assertNotEqual(self.run_script('deploy-k8s.sh', 'apply', 'staging', 'test-context').returncode, 0)
        self.assertFalse(any('castor.io/phase=application' in c for c in self.calls()))
        self.assertTrue(any('delete' in c and 'castor-release-lock' in c for c in self.calls()))

    def test_k8s_waits_for_init_before_app_and_uses_explicit_context(self):
        result = self.run_script('deploy-k8s.sh', 'apply', 'staging', 'test-context')
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = self.calls()
        wait = next(i for i, c in enumerate(calls) if 'wait' in c)
        app = next(i for i, c in enumerate(calls) if 'castor.io/phase=application' in c)
        self.assertLess(wait, app)
        for c in calls:
            if 'kustomize' not in c:
                self.assertEqual(c[1:3], ['--context', 'test-context'])

    def test_k8s_requires_context_before_contacting_cluster(self):
        self.assertNotEqual(self.run_script('deploy-k8s.sh', 'apply', 'staging').returncode, 0)
        self.assertEqual(self.calls(), [])

    def test_rollback_never_runs_old_initialization(self):
        for script, args in (
            ('deploy-compose.sh', ('rollback', 'full')),
            ('deploy-k8s.sh', ('rollback', 'staging', 'test-context')),
        ):
            result = self.run_script(script, *args)
            self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(any('run' in c and 'init-db' in c for c in self.calls()))
        self.assertFalse(any('create' in c and 'job' in c for c in self.calls()))

    def test_k8s_empty_secret_fails_before_cluster_mutation(self):
        (self.root / 'deploy/k8s/overlays/staging/secrets.env').write_text('CASTOR_JWT_KEY=\n')
        self.assertNotEqual(self.run_script('deploy-k8s.sh', 'apply', 'staging', 'test-context').returncode, 0)
        self.assertFalse(any('apply' in c or 'create' in c for c in self.calls()))


if __name__ == '__main__':
    unittest.main()

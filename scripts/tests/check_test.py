"""Dependency scan policy: only called vulnerabilities fail, accepted ones stop counting once fixed."""
import importlib.util
import json
from pathlib import Path
import sys
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'scripts'))
spec = importlib.util.spec_from_file_location('castor_check', ROOT / 'scripts/check.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def stream(*findings):
    # govulncheck writes indented objects one after another, not a JSON array
    messages = [{'config': {'scanner_name': 'govulncheck'}}, {'osv': {'id': 'GO-0000-0001'}}]
    messages += [{'finding': finding} for finding in findings]
    return '\n'.join(json.dumps(message, indent=2) for message in messages)


def called(osv, fixed=None):
    finding = {'osv': osv, 'trace': [{'module': 'example.com/m', 'function': 'Parse'}, {'module': 'castor'}]}
    if fixed:
        finding['fixed_version'] = fixed
    return finding


def imported_only(osv):
    return {'osv': osv, 'trace': [{'module': 'example.com/m', 'package': 'example.com/m'}]}


class VulnPolicyTest(unittest.TestCase):
    def test_only_findings_reaching_a_function_count(self):
        found = module.called_go_vulns(stream(imported_only('GO-1'), called('GO-2'), called('GO-3', 'v1.2.3')))
        self.assertEqual(found, {'GO-2': '', 'GO-3': 'v1.2.3'})

    def test_unreviewed_called_vulnerability_fails(self):
        problems, notes = module.vuln_problems({'GO-2': ''}, {})
        self.assertEqual(len(problems), 1)
        self.assertIn('ACCEPTED_GO_VULNS', problems[0])
        self.assertEqual(notes, [])

    def test_accepted_vulnerability_is_reported_not_failed(self):
        problems, notes = module.vuln_problems({'GO-2': ''}, {'GO-2': 'hardened'})
        self.assertEqual(problems, [])
        self.assertIn('hardened', notes[0])

    def test_accepted_vulnerability_fails_once_a_fix_exists(self):
        problems, _ = module.vuln_problems({'GO-2': 'v1.2.3'}, {'GO-2': 'hardened'})
        self.assertEqual(len(problems), 1)
        self.assertIn('v1.2.3', problems[0])

    def test_accepted_entries_name_their_reason(self):
        for osv, reason in module.ACCEPTED_GO_VULNS.items():
            self.assertRegex(osv, r'^GO-\d{4}-\d+$')
            self.assertTrue(reason.strip())


if __name__ == '__main__':
    unittest.main()

#!/usr/bin/env python3
"""Unified verification: developers before committing and AI agents after a change run the same checks.

Usage: check.py [all|api|web|scripts|vuln]...   (default all)

vuln is not part of all: the dependency vulnerability scan needs network access to fetch the
database and takes a while, so run check.py vuln before releases or periodically.
"""
import json
import os
import shutil
import sys

import _tooling as tooling

# Pinned so every run scans with the same upstream code.
GOVULNCHECK_VERSION = 'v1.8.0'
API = tooling.ROOT / 'apps/api'
# Vulnerabilities our code calls that were reviewed and accepted because upstream has no fix yet
# and the code is hardened against them. An entry stops counting as accepted once a fixed version
# is published: the scan then fails and asks for the upgrade.
ACCEPTED_GO_VULNS = {
    'GO-2026-6452': 'excelize panics on a crafted shared-string index; tabular.readXLSX recovers and '
                    'rejects the upload as unreadable (TestReadTurnsXLSXPanicsIntoUnreadable)',
}
WEB = tooling.ROOT / 'apps/web'


def step(title):
    colour = sys.stdout.isatty() and os.environ.get('NO_COLOR') is None
    print(f'\n\033[1;34m==> {title}\033[0m' if colour else f'\n==> {title}', flush=True)


def check_api():
    step('api: gofmt')
    unformatted = tooling.capture(['gofmt', '-l', '.'], cwd=API)
    if unformatted:
        raise tooling.Failure(f'unformatted files (run: cd apps/api && gofmt -w .):\n{unformatted}', code=1)
    step('api: go vet')
    tooling.run(['go', 'vet', './...'], cwd=API)
    # UTC like the production image, so tests depending on the local zone fail everywhere.
    # Windows ignores TZ; the tests themselves must stay zone independent there.
    env = dict(os.environ, TZ='UTC')
    # The race detector needs cgo, i.e. a C compiler (on Windows usually absent).
    race = tooling.capture(['go', 'env', 'CGO_ENABLED'], cwd=API) == '1'
    step('api: go test -race' if race else 'api: go test (no C compiler, so no -race)')
    tooling.run(['go', 'test', *(['-race'] if race else []), '-count=1', './...'], cwd=API, env=env)


def check_web():
    for title, command in (
        ('web: format', ['bun', 'run', 'format:check']),
        ('web: lint', ['bun', 'run', 'lint']),
        ('web: typecheck', ['bunx', 'tsc', '--noEmit']),
        ('web: test', ['bun', 'run', 'test']),
        ('web: build', ['bun', 'run', 'build']),
    ):
        step(title)
        tooling.run(command, cwd=WEB)


def check_scripts():
    # The Git hooks are the only shell scripts left; Git runs them with its own sh on every OS.
    if shutil.which('shellcheck'):
        step('scripts: shellcheck (Git hooks)')
        tooling.run(['shellcheck', '.husky/pre-commit', '.husky/pre-push'], cwd=tooling.ROOT)
    step('scripts: unittest')
    tooling.run([sys.executable, '-m', 'unittest', 'discover', '-s', 'scripts/tests', '-p', '*_test.py'],
                cwd=tooling.ROOT)


def called_go_vulns(stream):
    """Map each vulnerability that govulncheck's JSON stream traces into our code (a finding whose
    trace reaches a function, as its text mode reports) to its fixed version, '' when none exists."""
    decoder, position, called = json.JSONDecoder(), 0, {}
    while True:
        while position < len(stream) and stream[position].isspace():
            position += 1
        if position >= len(stream):
            return called
        message, position = decoder.raw_decode(stream, position)
        finding = message.get('finding')
        if finding and finding.get('trace') and finding['trace'][0].get('function'):
            called[finding['osv']] = finding.get('fixed_version', '')


def vuln_problems(called, accepted):
    """Return (problems, notes): unaccepted or now-fixable findings, and accepted ones to report."""
    problems, notes = [], []
    for osv, fixed in sorted(called.items()):
        info = f'https://pkg.go.dev/vuln/{osv}'
        if fixed:
            problems.append(f'{osv}: fixed in {fixed}, upgrade the module ({info})')
        elif osv in accepted:
            notes.append(f'{osv}: accepted, no upstream fix yet: {accepted[osv]}')
        else:
            problems.append(f'{osv}: called by our code and not reviewed ({info}); fix it, or harden '
                            f'the code and add it to ACCEPTED_GO_VULNS with the reason')
    return problems, notes


def check_vuln():
    step('vuln: go dependencies (govulncheck)')
    stream = tooling.capture(['go', 'run', f'golang.org/x/vuln/cmd/govulncheck@{GOVULNCHECK_VERSION}',
                              '-format', 'json', './...'], cwd=API)
    problems, notes = vuln_problems(called_go_vulns(stream), ACCEPTED_GO_VULNS)
    for note in notes:
        print(note)
    if problems:
        raise tooling.Failure('vulnerable code paths:\n' + '\n'.join(problems), code=1)
    step('vuln: web dependencies (bun audit)')
    tooling.run(['bun', 'audit'], cwd=WEB)


TARGETS = {
    'all': (check_api, check_web, check_scripts),
    'api': (check_api,),
    'web': (check_web,),
    'scripts': (check_scripts,),
    'vuln': (check_vuln,),
}


def main(argv):
    targets = argv or ['all']
    unknown = [target for target in targets if target not in TARGETS]
    if unknown:
        raise tooling.Failure(f'unknown target: {", ".join(unknown)} (choose from {"|".join(TARGETS)})')
    for target in targets:
        for check in TARGETS[target]:
            check()
    step('all checks passed')


if __name__ == '__main__':
    sys.exit(tooling.cli(main, sys.argv[1:]))

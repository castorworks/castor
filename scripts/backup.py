#!/usr/bin/env python3
"""Cold snapshots for Compose-managed volumes only. External/K8s data uses provider backups.

Usage: backup.py create <full|shared>
       backup.py list
       backup.py restore <full|shared> <snapshot> --confirm <snapshot>
Create/restore briefly stop this Compose project's services; previously running services restart.
shared snapshots every instance at once; stop the instances first, or back up a single
instance with compose-instance.py backup.
"""
import os
from pathlib import Path
import re
import sys
import tarfile

import _tooling as tooling

SERVICES = ('postgres', 'redis', 'rustfs')
APPLICATION = ('castor-api', 'castor-web', 'gateway')
HELPER_IMAGE = 'alpine:3.21'
STACKS = {
    'full': ('.env', ('compose.yaml', 'compose.infra.yaml'),
             ('.env', 'config', 'compose.yaml', 'compose.infra.yaml')),
    'shared': ('.env.shared', ('compose.infra.yaml', 'compose.shared.yaml'),
               ('.env.shared', 'compose.infra.yaml', 'compose.shared.yaml')),
}


def main(argv):
    action = tooling.arg(argv, 0, 'create|list|restore')
    backups = Path(os.environ.get('BACKUP_ROOT') or tooling.ROOT / 'deploy/backups')
    if action == 'list':
        for snapshot in tooling.snapshots(backups):
            print(snapshot)
        return
    stack = tooling.arg(argv, 1, 'full|shared')
    if action not in ('create', 'restore') or stack not in STACKS:
        raise tooling.Failure(__doc__)
    directory = tooling.ROOT / 'deploy/compose'
    env_file, files, config = STACKS[stack]
    dc = ['docker', 'compose', '--project-directory', str(directory), '--env-file', str(directory / env_file)]
    for name in files:
        dc += ['-f', str(directory / name)]
    if action == 'restore':
        name = tooling.arg(argv, 2, 'snapshot name')
        if not re.fullmatch(r'snapshot-[0-9]{8}_[0-9]{6}', name):
            raise tooling.Failure('snapshot name: snapshot-YYYYmmdd_HHMMSS')
        snapshot = backups / name
        if not (snapshot / 'complete').is_file():
            raise tooling.Failure(f'incomplete snapshot: {snapshot}', code=1)
        # Explicit operator acknowledgement; never infer restore consent.
        if argv[3:] != ['--confirm', name]:
            raise tooling.Failure(f'--confirm {name}')

    containers = [tooling.capture(dc + ['ps', '-aq', service]) for service in SERVICES]
    if not all(containers):
        raise tooling.Failure('every infrastructure service needs a container: ' + ', '.join(SERVICES), code=1)

    if action == 'create':
        snapshot = backups / f'snapshot-{tooling.timestamp()}'
        tooling.private_dir(snapshot, parents=True)

    # Pull the helper before interrupting services.
    tooling.run(['docker', 'pull', HELPER_IMAGE], stdout=tooling.DEVNULL)
    for service, container in zip(SERVICES, containers):
        image = tooling.capture(['docker', 'inspect', '--format', '{{.Image}}', container])
        if action == 'create':
            with tooling.private_file(snapshot / f'{service}.image') as output:
                output.write(image + '\n')
        else:
            archive = snapshot / f'{service}.tar'
            if image != (snapshot / f'{service}.image').read_text(encoding='utf-8').strip():
                raise tooling.Failure(f'{service}: image differs from the snapshot', code=1)
            if not archive.is_file() or not archive.stat().st_size:
                raise tooling.Failure(f'missing {archive}', code=1)
            with tarfile.open(archive) as members:
                members.getmembers()  # the whole archive must be readable before data is replaced

    running = tooling.capture(dc + ['ps', '--status', 'running', '--services']).splitlines()
    infra_running = [service for service in running if service in SERVICES]
    app_running = [service for service in running if service in APPLICATION]
    succeeded = False
    try:
        # Drain application requests before stopping the data services.
        for services in (app_running, infra_running):
            if services:
                tooling.run(dc + ['stop', *services])
        for service, container in zip(SERVICES, containers):
            data = '/var/lib/postgresql/data' if service == 'postgres' else '/data'
            if action == 'create':
                tooling.run(['docker', 'run', '--rm', '--volumes-from', f'{container}:ro',
                             '-v', f'{snapshot}:/backup', HELPER_IMAGE,
                             'tar', '-cf', f'/backup/{service}.tar', '-C', data, '.'])
            else:
                tooling.run(['docker', 'run', '--rm', '--volumes-from', container,
                             '-v', f'{snapshot}:/backup:ro', HELPER_IMAGE, 'sh', '-ec',
                             'find "$1" -mindepth 1 -maxdepth 1 -exec rm -rf {} \\; ; tar -xf "$2" -C "$1"',
                             'sh', data, f'/backup/{service}.tar'])
        if action == 'create':
            # Configuration is archived for inspection, never overwritten by a data restore.
            with tooling.private_file(snapshot / 'config.tar', 'wb') as output:
                with tarfile.open(fileobj=output, mode='w') as archive:
                    for name in config:
                        archive.add(directory / name, arcname=name)
            (snapshot / 'complete').touch()
            print(snapshot)
        succeeded = True
    finally:
        # A failed restore stays stopped so partially restored data is never served.
        if succeeded or action == 'create':
            for services in (infra_running, app_running):
                if services:
                    tooling.run(dc + ['start', '--wait', '--wait-timeout', '180', *services])


if __name__ == '__main__':
    sys.exit(tooling.cli(main, sys.argv[1:]))

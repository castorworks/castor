#!/usr/bin/env python3
"""Local and remote Compose deployment.

Local:  deploy-compose.py <up|pull|config|down|rollback> <full|external> [env-file]
        deploy-compose.py <up|pull|config|down> shared [env-file, default .env.shared]
        deploy-compose.py <up|pull|config|down|rollback> instance <env-file>
shared runs only the infrastructure; each instance joins it with its own database,
bucket and Redis namespace (see compose-instance.py).
Remote: deploy-compose.py remote <registry|tar> <host> <directory> <full|external> [archive]
Remote mode ships deploy/compose/.env and config/api.config.toml over SSH and runs
docker compose there; the remote host needs only Docker and tar. Builds are separate.
"""
import os
import posixpath
import re
import shlex
import shutil
import sys
import tarfile

import _tooling as tooling

STACK_FILES = {
    'full': ('compose.yaml', 'compose.infra.yaml'),
    'external': ('compose.yaml',),
    'shared': ('compose.infra.yaml', 'compose.shared.yaml'),
    'instance': ('compose.yaml', 'compose.instance.yaml'),
}
APPLICATION = ('castor-api', 'castor-web', 'gateway')
INFRASTRUCTURE = ('postgres', 'redis', 'rustfs')
WAIT = ('--wait', '--wait-timeout', '180')
# Shipped to remote hosts, relative to the repository root.
REMOTE_FILES = ('deploy/compose/compose.yaml', 'deploy/compose/compose.infra.yaml',
                'deploy/compose/config/nginx.conf', 'deploy/compose/config/redis.conf',
                'deploy/compose/config/api.config.toml', 'deploy/compose/.env')


def compose_command(directory, env_file, stack, join=os.path.join):
    """docker compose for a stack; join builds paths on the machine that runs it."""
    command = ['docker', 'compose', '--project-directory', str(directory), '--env-file', str(env_file)]
    for name in STACK_FILES[stack]:
        command += ['-f', join(str(directory), name)]
    return command


def deploy(dc, action, stack, env_file=None):
    """One Compose action; dc(args) runs docker compose with the stack's files."""
    if action == 'config':
        dc(['--profile', 'tools', 'config', '--quiet'])
    elif action == 'pull':
        dc(['--profile', 'tools', 'pull'])
    elif action == 'down':
        dc(['--profile', 'tools', 'down'])
    elif action == 'rollback':
        if stack == 'shared':
            raise tooling.Failure('shared has no application to roll back')
        # Operator has selected database-compatible images/config; never run old migrations.
        dc(['up', '-d', '--force-recreate', *WAIT, *APPLICATION])
    elif action == 'up':
        dc(['--profile', 'tools', 'config', '--quiet'])
        if stack in ('full', 'shared'):
            dc(['up', '-d', *WAIT, *INFRASTRUCTURE])
        if stack == 'full':
            dc(['run', '--rm', '--no-deps', 'rustfs-init'])
        # Buckets belong to instances; provision creates each one with its own user.
        if stack == 'instance':
            code = tooling.load_script('compose-instance').main(['provision', str(env_file)])
            if code:
                raise tooling.Failure(code=code)
        if stack != 'shared':
            # Always rerun for the requested image/config, even if an earlier run succeeded.
            dc(['run', '--rm', '--no-deps', 'init-db'])
            dc(['up', '-d', '--force-recreate', *WAIT, *APPLICATION])
    else:
        raise tooling.Failure(__doc__)


def remote(argv):
    mode = tooling.arg(argv, 1, 'registry|tar')
    host = tooling.arg(argv, 2, 'SSH host')
    dest = tooling.arg(argv, 3, 'remote directory')
    stack = tooling.arg(argv, 4, 'full|external')
    archive = tooling.arg(argv, 5, 'archive path') if mode == 'tar' else None
    if mode not in ('registry', 'tar') or stack not in ('full', 'external'):
        raise tooling.Failure(__doc__)
    # An absolute POSIX path safe inside single quotes; a host that is not an ssh option.
    if not re.fullmatch(r'/[a-zA-Z0-9_./-]+', dest) or host.startswith('-'):
        raise tooling.Failure('remote directory must be an absolute path of [a-zA-Z0-9_./-]; host must not start with -')
    directory = tooling.ROOT / 'deploy/compose'
    deploy(lambda args: tooling.run(compose_command(directory, directory / '.env', stack) + args,
                                    stdout=tooling.DEVNULL), 'config', stack)

    def ssh(command, **kwargs):
        return tooling.run(['ssh', host, command], **kwargs)

    def ship(stdin):
        with tarfile.open(fileobj=stdin, mode='w|') as bundle:
            for name in REMOTE_FILES:
                info = bundle.gettarinfo(tooling.ROOT / name, arcname=name)
                info.mode, info.uid, info.gid, info.uname, info.gname = 0o644, 0, 0, '', ''
                with open(tooling.ROOT / name, 'rb') as source:
                    bundle.addfile(info, source)

    quoted = shlex.quote(dest)
    ssh(f'umask 077; mkdir -p {quoted}/deploy/compose/config')
    tooling.feed(['ssh', host, f'umask 077; tar -xf - -C {quoted}'], ship)
    # The API runs as UID 10001. The parent directory stays private to the deploy user.
    ssh(f'chmod 600 {quoted}/deploy/compose/.env; chmod 644 {quoted}/deploy/compose/config/api.config.toml')
    if archive:
        # docker load reads gzip directly: ship the archive as is.
        with open(archive, 'rb') as source:
            tooling.feed(['ssh', host, 'docker load'], lambda stdin: shutil.copyfileobj(source, stdin))
    remote_dir = posixpath.join(dest, 'deploy/compose')
    compose = compose_command(remote_dir, posixpath.join(remote_dir, '.env'), stack, posixpath.join)

    def dc(args):
        ssh(shlex.join(compose + args))

    if mode == 'registry':
        deploy(dc, 'pull', stack)
    deploy(dc, 'up', stack)


def main(argv):
    action = tooling.arg(argv, 0, 'up|pull|config|down|rollback|remote')
    if action == 'remote':
        return remote(argv)
    stack = tooling.arg(argv, 1, 'full|external|shared|instance')
    if stack not in STACK_FILES:
        raise tooling.Failure(__doc__)
    directory = tooling.ROOT / 'deploy/compose'
    if stack == 'instance':
        env_file = tooling.arg(argv, 2, 'instance env file')
    else:
        env_file = argv[2] if len(argv) > 2 else directory / ('.env.shared' if stack == 'shared' else '.env')
    command = compose_command(directory, env_file, stack)
    deploy(lambda args: tooling.run(command + args), action, stack, env_file)


if __name__ == '__main__':
    sys.exit(tooling.cli(main, sys.argv[1:]))

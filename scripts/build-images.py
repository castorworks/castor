#!/usr/bin/env python3
"""Build the API and Web images.

Usage: build-images.py <load|push|tar> <image-prefix> <tag> [archive]
Example: build-images.py push registry.example.com/castor v1.0.0
PLATFORM overrides the target platform (default linux/amd64). GOPROXY, UBUNTU_MIRROR and
UBUNTU_SECURITY_MIRROR, when set in the environment, are passed to the API build as build
arguments (mirrors for build machines that cannot reach the official sources).
"""
import gzip
import os
from pathlib import Path
import shutil
import sys

import _tooling as tooling

# Build arguments each Dockerfile accepts from the environment; unset ones keep the Dockerfile default.
BUILD_ARG_ENV = {'api': ('GOPROXY', 'UBUNTU_MIRROR', 'UBUNTU_SECURITY_MIRROR'), 'web': ()}


def build_args(app, environ):
    """--build-arg flags for the variables of BUILD_ARG_ENV[app] that are set."""
    flags = []
    for name in BUILD_ARG_ENV[app]:
        if environ.get(name):
            flags += ['--build-arg', f'{name}={environ[name]}']
    return flags


def main(argv):
    mode = tooling.arg(argv, 0, 'load|push|tar')
    if mode not in ('load', 'push', 'tar'):
        raise tooling.Failure(__doc__)
    prefix = tooling.arg(argv, 1, 'image-prefix').rstrip('/')
    tag = tooling.arg(argv, 2, 'tag')
    archive = Path(tooling.arg(argv, 3, 'archive path')) if mode == 'tar' else None
    images = [f'{prefix}/castor-{app}:{tag}' for app in ('api', 'web')]
    for app, image in zip(('api', 'web'), images):
        context = tooling.ROOT / 'apps' / app
        tooling.run(['docker', 'buildx', 'build', '--platform', os.environ.get('PLATFORM', 'linux/amd64'),
                     '--push' if mode == 'push' else '--load', *build_args(app, os.environ),
                     '-f', str(context / 'Dockerfile'),
                     '-t', image, str(context)])
    if archive:
        archive.parent.mkdir(parents=True, exist_ok=True)
        with gzip.open(archive, 'wb') as target:
            tooling.drain(['docker', 'save', *images], lambda source: shutil.copyfileobj(source, target))


if __name__ == '__main__':
    sys.exit(tooling.cli(main, sys.argv[1:]))

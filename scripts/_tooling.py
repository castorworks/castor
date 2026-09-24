"""Shared helpers for the repository scripts; standard library only, Python 3.9+.

The scripts run unchanged on Windows, macOS and Linux, so they never go through a shell:
every external tool is started directly with an argument list (secrets reach tools on
stdin, never argv), and pipes, archives and random secrets are handled in Python.
"""
import importlib.util
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]
PIPE = subprocess.PIPE
DEVNULL = subprocess.DEVNULL


class Failure(Exception):
    """Stop the script with an exit status and an optional message on stderr."""

    def __init__(self, message='', code=2):
        super().__init__(message)
        self.code = code


def cli(main, argv):
    """Run main(argv) and turn failures into exit statuses instead of tracebacks."""
    try:
        return main(argv) or 0
    except Failure as error:
        if str(error):
            print(error, file=sys.stderr)
        return error.code
    except subprocess.CalledProcessError as error:
        return error.returncode or 1
    except KeyboardInterrupt:
        return 130


def arg(argv, index, name):
    """Positional argument, or exit 2 naming what is missing (like ${1:?name})."""
    if len(argv) <= index or not argv[index]:
        raise Failure(f'missing argument: {name}')
    return argv[index]


def process(args, **kwargs):
    """Start a tool. PATH lookup honours PATHEXT, so docker.exe or bun.cmd are found on Windows."""
    return subprocess.Popen([shutil.which(args[0]) or args[0], *args[1:]], **kwargs)


def _checked(proc, args, check=True):
    if check and proc.returncode:
        raise subprocess.CalledProcessError(proc.returncode, args)
    return proc.returncode


def run(args, *, input=None, stdin=None, stdout=None, check=True, **kwargs):
    """Run a tool to completion; input (bytes) is written to its stdin."""
    proc = process(args, stdin=PIPE if input is not None else stdin, stdout=stdout, **kwargs)
    proc.communicate(input)
    return _checked(proc, args, check)


def capture(args, **kwargs):
    """Run a tool and return its stdout as stripped text."""
    proc = process(args, stdout=PIPE, **kwargs)
    output, _ = proc.communicate()
    _checked(proc, args)
    return output.decode('utf-8').strip()


def feed(args, write, **kwargs):
    """Run a tool while write(stdin) streams into it."""
    proc = process(args, stdin=PIPE, **kwargs)
    try:
        write(proc.stdin)
    except BrokenPipeError:
        pass  # the tool exited early; its status below explains why
    finally:
        try:
            proc.stdin.close()
        except BrokenPipeError:
            pass
    proc.wait()
    return _checked(proc, args)


def drain(args, read, **kwargs):
    """Run a tool while read(stdout) consumes its output."""
    proc = process(args, stdout=PIPE, **kwargs)
    try:
        read(proc.stdout)
    finally:
        proc.stdout.close()
        proc.wait()
    return _checked(proc, args)


def env_get(path, key):
    """The last KEY=value assignment in a Compose env file, verbatim ('' when absent)."""
    value = ''
    for line in Path(path).read_text(encoding='utf-8').splitlines():
        if line.startswith(key + '='):
            value = line[len(key) + 1:]
    return value


def private_dir(path, parents=False):
    """Create a directory only the current user can read (POSIX modes; NTFS keeps its ACLs)."""
    if parents:
        Path(path).parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    Path(path).mkdir(mode=0o700)


def private_file(path, mode='w'):
    """Open a new file with owner-only permissions; fails if it already exists."""
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, 'O_BINARY', 0), 0o600)
    return os.fdopen(descriptor, mode, **({} if 'b' in mode else {'encoding': 'utf-8', 'newline': '\n'}))


def timestamp():
    return time.strftime('%Y%m%d_%H%M%S')


def snapshots(directory):
    """Snapshot directories under a backup root, oldest first."""
    directory = Path(directory)
    return sorted(p for p in directory.glob('snapshot-*') if p.is_dir()) if directory.is_dir() else []


def load_script(name):
    """Import scripts/<name>.py (hyphenated file names are not importable directly)."""
    path = Path(__file__).with_name(name + '.py')
    spec = importlib.util.spec_from_file_location('castor_' + name.replace('-', '_'), path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module

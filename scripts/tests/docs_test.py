"""Human-facing docs ship in English, Simplified Chinese, Japanese and Korean and must not drift.

A translation may reword prose, but its structure has to match the English original:
same headings, same shell commands, same tables, same links (modulo the language suffix).
"""
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[2]

# (suffix, name shown in the language switcher); '' is the English default file.
LANGS = [('', 'English'), ('zh-CN', '简体中文'), ('ja', '日本語'), ('ko', '한국어')]
DOCS = ['README.md', 'SECURITY.md', 'CHANGELOG.md', 'deploy/README.md']
LANG_SUFFIX = re.compile(r'\.(zh-CN|ja|ko)\.md\b')


def variant(doc, suffix):
    path = ROOT / doc
    return path if not suffix else path.with_name(f'{path.stem}.{suffix}.md')


def switcher(doc, current):
    stem = Path(doc).stem
    return ' | '.join(f'**{name}**' if suffix == current
                      else f'[{name}]({stem}{"." + suffix if suffix else ""}.md)'
                      for suffix, name in LANGS)


def outline(text):
    """What a translation must preserve."""
    headings, commands, tables, links = [], [], [], []
    fence = None
    for line in text.splitlines():
        if line.startswith('```'):
            fence = None if fence is not None else line[3:].strip()
            continue
        if fence is not None:
            if fence == 'bash':
                # Comments are translated; the commands themselves must be identical.
                command = re.sub(r'(^|\s+)#.*$', '', line).rstrip()
                if command:
                    commands.append(command)
            continue
        if heading := re.match(r'^(#+) ', line):
            headings.append(len(heading.group(1)))
        if line.startswith('|'):
            tables.append(line.count('|'))
        links += [LANG_SUFFIX.sub('.md', target)
                  for target in re.findall(r'\]\(([^)#\s]+)', line)
                  if not target.startswith('http')]
    return {'heading levels': headings, 'bash commands': commands,
            'table rows': tables, 'links': sorted(links)}


class DocsTest(unittest.TestCase):
    def test_every_doc_has_all_languages_with_matching_structure(self):
        for doc in DOCS:
            original = outline(variant(doc, '').read_text(encoding='utf-8'))
            for suffix, _ in LANGS[1:]:
                path = variant(doc, suffix)
                with self.subTest(doc=str(path.relative_to(ROOT))):
                    self.assertTrue(path.exists(), 'missing translation')
                    translated = outline(path.read_text(encoding='utf-8'))
                    for key, expected in original.items():
                        self.assertEqual(translated[key], expected, key)

    def test_language_switcher_links_every_version(self):
        for doc in DOCS:
            for suffix, _ in LANGS:
                path = variant(doc, suffix)
                if not path.exists():
                    continue  # reported by the structure test
                with self.subTest(doc=str(path.relative_to(ROOT))):
                    lines = path.read_text(encoding='utf-8').splitlines()
                    self.assertEqual(lines[2], switcher(doc, suffix))

    def test_translations_link_to_their_own_language(self):
        for doc in DOCS:
            for suffix, _ in LANGS[1:]:
                path = variant(doc, suffix)
                if not path.exists():
                    continue
                text = path.read_text(encoding='utf-8').split('\n', 3)[3]  # skip the switcher
                for target in re.findall(r'\]\(([^)#\s]+\.md)\)', text):
                    other = (path.parent / target).resolve()
                    if other.relative_to(ROOT).as_posix() in DOCS:
                        with self.subTest(doc=str(path.relative_to(ROOT)), link=target):
                            self.fail(f'link the {suffix} version instead')

    def test_relative_links_resolve(self):
        for doc in DOCS:
            for suffix, _ in LANGS:
                path = variant(doc, suffix)
                if not path.exists():
                    continue
                for target in re.findall(r'\]\(([^)#\s]+)', path.read_text(encoding='utf-8')):
                    if not target.startswith('http'):
                        with self.subTest(doc=str(path.relative_to(ROOT)), link=target):
                            self.assertTrue((path.parent / target).exists())


if __name__ == '__main__':
    unittest.main()

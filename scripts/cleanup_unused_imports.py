"""
Phase 24.2 cleanup helper — strips unused imports flagged by svelte-check.

Reads `/tmp/svelte-errors.txt` (output of `npx svelte-check --threshold error`),
parses each "X is declared but its value is never read" error, and removes
that name from named-import braces. Default and namespace imports are left
alone so human review can decide whether they're side-effect imports.

Idempotent: running twice changes nothing on the second run.
"""

import re
import os
from collections import defaultdict

ERROR_RE = re.compile(r'ERROR "([^"]+)" (\d+):(\d+) "([^"]+)"')
UNUSED_RE = re.compile(r"'([^']+)' is declared but its value is never read\.")

ROOT = "frontend"

errors_by_file = defaultdict(list)
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
with open(os.path.join(SCRIPT_DIR, 'svelte-errors.txt'), 'r', encoding='utf-8') as f:
    for line in f:
        m = ERROR_RE.search(line)
        if not m:
            continue
        file, lineno, col, msg = m.groups()
        errors_by_file[file].append({'line': int(lineno), 'col': int(col), 'msg': msg})

NAMED_IMPORT_RE = re.compile(
    r'(import\s+(?:[A-Za-z_$][\w$]*\s*,\s*)?\{)([^}]+)(\}\s*from\s*[\'"][^\'"]+[\'"]\s*;?)',
    re.MULTILINE,
)


def strip_from_named_imports(text: str, name: str) -> str:
    def repl(m: re.Match) -> str:
        head, inner, tail = m.group(1), m.group(2), m.group(3)
        parts = [p.strip() for p in inner.split(',')]
        kept = [p for p in parts if p and p.split(' as ')[0].strip() != name]
        if len(kept) == len(parts):
            return m.group(0)
        if not kept:
            # Whole named-import block became empty.
            # If a default import preceded the braces, keep it.
            head_stripped = head.rstrip().rstrip('{').rstrip()
            if head_stripped.endswith(','):
                head_stripped = head_stripped[:-1].rstrip()
            if head_stripped == 'import':
                # No default import → drop the line entirely.
                return ''
            # Default import remains → reconstruct without braces.
            from_part = re.search(r"from\s*['\"][^'\"]+['\"]\s*;?", tail)
            return f"{head_stripped} {from_part.group(0) if from_part else ''}"
        return head + ' ' + ', '.join(kept) + ' ' + tail
    return NAMED_IMPORT_RE.sub(repl, text)


fixed_imports = 0
files_changed = 0

for relpath, errors in errors_by_file.items():
    path = os.path.join(ROOT, relpath.replace('\\', os.sep))
    if not os.path.exists(path):
        continue

    unused = set()
    for err in errors:
        m = UNUSED_RE.search(err['msg'])
        if m:
            unused.add(m.group(1))

    if not unused:
        continue

    with open(path, 'r', encoding='utf-8') as f:
        src = f.read()
    original = src

    for name in unused:
        new_src = strip_from_named_imports(src, name)
        if new_src != src:
            src = new_src
            fixed_imports += 1

    # Clean up any empty named-import lines we left behind.
    src = re.sub(
        r'import\s*\{\s*\}\s*from\s*[\'"][^\'"]+[\'"]\s*;?\n?',
        '',
        src,
    )

    if src != original:
        with open(path, 'w', encoding='utf-8') as f:
            f.write(src)
        files_changed += 1

print(f"Fixed {fixed_imports} unused-import deletions across {files_changed} files")

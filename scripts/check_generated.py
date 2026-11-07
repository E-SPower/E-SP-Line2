#!/usr/bin/env python3
"""检查生成物是否与 matrix.json 一致。

改了 matrix.json 却忘了重跑生成脚本时，测试会在这里失败。
用法：python3 scripts/check_generated.py matrix.json
"""

import pathlib
import subprocess
import sys
import tempfile

MATRIX = pathlib.Path(sys.argv[1])
ROOT = pathlib.Path(__file__).resolve().parent.parent

TARGETS = [
    ('gen_platformspec.py', 'internal/platformspec'),
    ('gen_capabilities.py', 'docs/developer-guide/platform-capabilities.md'),
]


def run(script, out):
    subprocess.run([sys.executable, str(ROOT / 'scripts' / script), str(MATRIX), str(out)], check=True)


def main():
    failed = False
    for script, target in TARGETS:
        actual = ROOT / target
        if actual.is_dir():
            tmp = pathlib.Path(tempfile.mkdtemp())
            run(script, tmp)
            for generated in sorted(tmp.glob('*.go')):
                if not (generated.name == 'commands.go' or generated.name.startswith('platform_')):
                    continue
                current = actual / generated.name
                if not current.exists() or current.read_text() != generated.read_text():
                    print(f'stale: {current.relative_to(ROOT)}')
                    failed = True
            extra = {p.name for p in actual.glob('*.go') if p.name == 'commands.go' or p.name.startswith('platform_')} - {p.name for p in tmp.glob('*.go')}
            for name in sorted(extra):
                print(f'stale extra file: {(actual / name).relative_to(ROOT)}')
                failed = True
        else:
            with tempfile.TemporaryDirectory() as tmp:
                out = pathlib.Path(tmp) / 'out.md'
                run(script, out)
                if not actual.exists() or actual.read_text() != out.read_text():
                    print(f'stale: {target}')
                    failed = True
    if failed:
        print('\nrerun the generators listed in docs/developer-guide/catbus-migration.md')
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())

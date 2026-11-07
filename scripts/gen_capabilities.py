#!/usr/bin/env python3
"""从能力矩阵生成 docs/developer-guide/platform-capabilities.md。

与 gen_platformspec.py 同源：先有 matrix.json，再长出 Go 定义和这份文档。

用法：
  python3 scripts/gen_capabilities.py matrix.json docs/developer-guide/platform-capabilities.md
"""

import json
import pathlib
import sys

MATRIX = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding='utf-8'))
OUT = pathlib.Path(sys.argv[2])

ITEM = {'xhs': '笔记', 'douyin': '作品', 'tiktok': '视频', 'bilibili': '稿件', 'kuaishou': '作品',
        'weibo': '微博', 'xianyu': '闲置商品', 'taobao': '商品', 'jd': '商品 SKU', 'x': '推文',
        'youtube': '视频', '12306': '车次', 'jimeng': '作品', 'maoyan': '演出', 'mteam': '种子资源'}

UPSTREAM_DOC = {
    'xhs': 'https://github.com/cv-cat/Spider_XHS',
    'douyin': 'https://github.com/cv-cat/DouYin_Spider',
    'tiktok': 'https://github.com/cv-cat/TiktokApis',
    'bilibili': 'https://github.com/cv-cat/BilibiliApis',
    'kuaishou': 'https://github.com/cv-cat/KuaiShou-Spider',
    'weibo': 'https://github.com/cv-cat/WeiboApis',
    'xianyu': 'https://github.com/cv-cat/XianYuApis',
    'taobao': 'https://github.com/cv-cat/TaoBaoApis',
    'jd': 'https://github.com/cv-cat/JdApis',
    'x': 'https://github.com/cv-cat/XApis',
}

SYMBOL = {'implemented'  : '✓', 'partial': '◐', 'planned': '○', 'unsupported': '—'}

HEADER = '''# Platform capability matrix

Per-platform web capability support. Migrated from catbus `docs/capabilities.md` (MIT); see the [catbus migration notes](./catbus-migration.md).

- Generated from the capability matrix by `scripts/gen_capabilities.py`; **do not hand-edit**. A test checks that it is current.
- The same matrix feeds `internal/platformspec`; the command tree, adapter manifests and `GET /api/v1/platforms/spec` all derive from it.
- The app and pc endpoints are all planned and are not listed.

| Symbol | Meaning | Support | Runtime result |
|---|---|---|---|
| ✓ | upstream has it | `implemented` | runs |
| ◐ | upstream has it in part; the limitation is in the platform notes | `partial` | runs |
| ○ | the platform has the concept, upstream does not; planned | `planned` | `NOT_IMPLEMENTED`, exit code 4 |
| — | the platform has no such concept | `unsupported` | `UNSUPPORTED`, exit code 2 |

Platforms come in two stages: **active** ({active}) and **upcoming** ({upcoming}). Upcoming platforms register as `inactive` adapters while keeping their rows.
'''

SECTION = {
    'notice 与 msg': 'notice and msg',
    'media、folder、series、history、topic、poi': 'media, folder, series, history, topic, poi',
    'product 与商品评价': 'product and reviews',
    'feed 与 keyword': 'feed and keyword',
}

def table(platforms):
    return '| 命令 | ' + ' | '.join(platforms) + ' |\n|' + '---|' * (len(platforms) + 1)


def row(name, cells, platforms):
    return '| `' + name + '` | ' + ' | '.join(SYMBOL.get(cells.get(p, 'unsupported'), '—') for p in platforms) + ' |'


def main():
    platforms = MATRIX['platforms']
    upcoming = [p for p in platforms if p in ('jimeng', 'maoyan', 'mteam')]
    active = [p for p in platforms if p not in upcoming]

    by_section = {}
    for c in MATRIX['commands']:
        by_section.setdefault(c['section'], []).append(c)

    chunks = [HEADER.format(active=' · '.join(active), upcoming=' · '.join(upcoming))]

    for scope, group in (('Active', active), ('Upcoming', upcoming)):
        chunks.append(f'## {scope}\n')
        for section, cmds in by_section.items():
            # 只列出这一批平台里真的有内容的命令，整节都空就跳过，免得满屏的 —。
            cmds = [c for c in cmds if any(c['platforms'].get(p) not in (None, 'unsupported') for p in group)]
            if not cmds:
                continue
            chunks.append(f'### {SECTION.get(section, section)}\n')
            chunks.append(table(group))
            for c in cmds:
                chunks.append(row(name_of(c), c['platforms'], group))
            chunks.append('')
            for c in cmds:
                for n in MATRIX['notes']:
                    if n['command'] == name_of(c) and n['platform'] in group:
                        chunks.append(f'- {n["platform"]} `{n["command"]}`: {n["text"]}')
            chunks.append('')

    chunks.append('## Platforms\n')
    for p in MATRIX['platforms']:
        stage = 'upcoming' if p in upcoming else 'active'
        item = ITEM.get(p, '')
        url = UPSTREAM_DOC.get(p)
        link = f' · [docs]({url})' if url else ''
        chunks.append(f'- `{p}` — {item} ({stage}){link}')
    chunks.append('')
    chunks.append('## Inline notes\n')
    for n in MATRIX['notes']:
        who = 'all' if n['platform'] == '*' else n['platform']
        chunks.append(f"- {who} `{n['command']}`: {n['text']}")

    chunks.append('\n## Login methods\n')
    chunks.append(table(platforms))
    methods = sorted({m for p in MATRIX['login'].values() for m in p}) or ['cookie']
    for m in methods:
        chunks.append('| ' + m + ' | ' + ' | '.join('✓' if m in MATRIX['login'].get(p, []) else '' for p in platforms) + ' |')

    OUT.write_text('\n'.join(chunks) + '\n', encoding='utf-8')


def name_of(c):
    return ' '.join(x for x in (c['resource'], c['action']) if x)


if __name__ == '__main__':
    main()

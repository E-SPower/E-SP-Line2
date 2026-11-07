#!/usr/bin/env python3
"""从 catbus 能力矩阵生成 internal/platformspec。

catbus 的命令解析、`platforms` 命令、能力矩阵都由同一份平台注册表生成；这里把导出的
matrix.json 连同平台元数据转成 Go 定义，避免手抄出错。

用法：
  python3 scripts/gen_platformspec.py matrix.json internal/platformspec
"""

import json
import pathlib
import shutil
import subprocess
import sys

OUT = pathlib.Path(sys.argv[2] if len(sys.argv) > 2 else 'internal/platformspec')
MATRIX = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding='utf-8'))

# 上游来源（对应 catbus 的 src/platforms/<p>/UPSTREAM）。
META = {
    'xhs': dict(name='小红书', name_en='Xiaohongshu', item='笔记', item_en='note', aliases=['xiaohongshu', 'rednote'],
                upstream='https://github.com/cv-cat/Spider_XHS', site='https://www.xiaohongshu.com',
                dm='https://www.xiaohongshu.com/messages'),
    'douyin': dict(name='抖音', name_en='Douyin', item='作品', item_en='work', aliases=['dy'],
                   upstream='https://github.com/cv-cat/DouYin_Spider', site='https://www.douyin.com',
                   dm='https://www.douyin.com/messages'),
    'tiktok': dict(name='TikTok', name_en='TikTok', item='视频', item_en='video', aliases=['tt'],
                   upstream='https://github.com/cv-cat/TiktokApis', site='https://www.tiktok.com',
                   dm='https://www.tiktok.com/messages'),
    'bilibili': dict(name='B 站', name_en='Bilibili', item='稿件', item_en='video', aliases=['bili'],
                     upstream='https://github.com/cv-cat/BilibiliApis', site='https://www.bilibili.com',
                     dm='https://message.bilibili.com'),
    'kuaishou': dict(name='快手', name_en='Kuaishou', item='作品', item_en='work', aliases=['ks'],
                     upstream='https://github.com/cv-cat/KuaiShou-Spider', site='https://www.kuaishou.com',
                     dm='https://www.kuaishou.com/message'),
    'weibo': dict(name='微博', name_en='Weibo', item='微博', item_en='post', aliases=['wb'],
                  upstream='https://github.com/cv-cat/WeiboApis', site='https://m.weibo.cn',
                  dm='https://m.weibo.cn/message'),
    'xianyu': dict(name='闲鱼', name_en='Xianyu', item='闲置商品', item_en='listing', aliases=['goofish', 'xy'],
                   upstream='https://github.com/cv-cat/XianYuApis', site='https://www.goofish.com',
                   dm='https://www.goofish.com/im'),
    'taobao': dict(name='淘宝', name_en='Taobao', item='商品', item_en='product', aliases=['tb'],
                   upstream='https://github.com/cv-cat/TaoBaoApis', site='https://www.taobao.com',
                   dm='https://www.taobao.com'),
    'jd': dict(name='京东', name_en='JD', item='商品 SKU', item_en='SKU', aliases=['jingdong'],
               upstream='https://github.com/cv-cat/JdApis', site='https://www.jd.com',
               dm='https://dongdong.jd.com'),
    'x': dict(name='X', name_en='X', item='推文', item_en='post', aliases=['twitter'],
              upstream='https://github.com/cv-cat/XApis', site='https://x.com',
              dm='https://x.com/messages'),
    'youtube': dict(name='YouTube', name_en='YouTube', item='视频', item_en='video', aliases=['yt'],
                    upstream='https://www.youtube.com', site='https://www.youtube.com',
                    dm='https://studio.youtube.com'),
    '12306': dict(name='12306 铁路', name_en='China Railway 12306', item='车次', item_en='train', aliases=[],
                  upstream='https://kyfw.12306.cn', site='https://kyfw.12306.cn', dm=''),
    'jimeng': dict(name='即梦', name_en='Jimeng', item='作品', item_en='work', aliases=['dreamina'],
                   upstream='https://jimeng.jianying.com', site='https://jimeng.jianying.com', dm=''),
    'maoyan': dict(name='猫眼演出', name_en='Maoyan Shows', item='演出', item_en='show', aliases=[],
                   upstream='https://yanchu.maoyan.com', site='https://yanchu.maoyan.com', dm=''),
    'mteam': dict(name='M-Team', name_en='M-Team', item='种子资源', item_en='torrent',
                  aliases=['mt', 'm-team', 'mantou'],
                  upstream='https://api.m-team.cc/api', site='https://kp.m-team.cc',
                  dm='https://kp.m-team.cc/messages'),
}

COOKIE = [
    ('cookie', 'string', True, '平台网页版登录 Cookie', 'Platform web login cookie'),
    ('device_id', 'string', False, '设备标识（可选）', 'Device ID (optional)'),
]
CONFIG = {
    '12306': [],
    'jimeng': [('token', 'string', True, '即梦 CLI 登录态（dreamina relogin 写进钥匙串的 JSON）', 'Jimeng CLI credential JSON')],
    'mteam': [
        ('api_key', 'string', True, 'M-Team API Key', 'M-Team API key'),
        ('uid', 'string', False, '站点用户 ID（API Key 模式下必填）', 'Site user id (required with API key)'),
        ('box', 'string', False, '默认站内信箱：inbox / system / sent', 'Default mailbox: inbox / system / sent'),
    ],
}
for _p in META:
    CONFIG.setdefault(_p, list(COOKIE))
for _p in list(CONFIG):
    if _p not in ('12306', 'jimeng', 'mteam'):
        CONFIG[_p] += [
            ('heartbeat_interval', 'integer', False, '心跳间隔（秒）', 'Heartbeat interval in seconds'),
            ('reconnect_delay', 'integer', False, '重连延迟（秒）', 'Reconnect delay in seconds'),
        ]

DEFAULTS = {'heartbeat_interval': 15, 'reconnect_delay': 5}

STAGES = {
    'xhs': 'Active', 'douyin': 'Active', 'tiktok': 'Active', 'bilibili': 'Active',
    'kuaishou': 'Active', 'weibo': 'Active', 'xianyu': 'Active', 'taobao': 'Active',
    'jd': 'Active', 'x': 'Active', 'youtube': 'Active', '12306': 'Active',
    'jimeng': 'Upcoming', 'maoyan': 'Upcoming', 'mteam': 'Upcoming',
}


def gostr(s):
    return json.dumps(s, ensure_ascii=False)


def ident(p):
    out = p.replace('-', '')
    return 'P' + out if out[0].isdigit() else out.capitalize()


def support(commands, platform, resource, action):
    for c in commands:
        if c['resource'] == resource and c['action'] == action:
            return c['platforms'][platform]
    return 'unsupported'


def capabilities(commands, platform):
    """把能力矩阵折算成能力清单：保留既有词表，另加 `<resource>.<action>` 细项。"""
    caps = set()

    def ok(res, act):
        return support(commands, platform, res, act) in ('implemented', 'partial')

    if ok('msg', 'listen'):
        caps.add('receive_message')
    if ok('msg', 'send'):
        caps.update(('send_text', 'send_image', 'create_conversation'))
    if ok('msg', 'history'):
        caps.add('get_history')
    if ok('media', 'upload') or ok('item', 'publish'):
        caps.add('upload_media')
    if ok('item', 'publish'):
        caps.update(('send_image', 'send_video'))
    if ok('item', 'get') and platform in ('taobao', 'xianyu', 'jd', 'douyin'):
        caps.add('get_product_info')
    if ok('order', 'list'):
        caps.add('get_order_info')
    if ok('auth', 'status'):
        caps.add('refresh_token')

    for c in commands:
        state = c['platforms'][platform]
        if state == 'unsupported':
            continue
        name = f"{c['resource']}.{c['action']}"
        caps.add(name)
        if state == 'planned':
            caps.add(name + '.planned')
    return sorted(caps)


def write_platform(p, commands, notes, ordinal):
    meta = META[p]
    out = [
        '// Code generated by scripts/gen_platformspec.py; DO NOT EDIT.',
        '',
        'package platformspec',
        '',
        f'func init() {{ register(platform{ident(p)}()) }}',
        '',
        f'func platform{ident(p)}() *Platform {{',
        '\treturn &Platform{',
        f'\t\tID: {gostr(p)},',
        f'\t\tOrdinal: {ordinal},',
        f'\t\tName: {gostr(meta["name"])},',
        f'\t\tNameEN: {gostr(meta["name_en"])},',
        f'\t\tItem: {gostr(meta["item"])},',
        f'\t\tItemEN: {gostr(meta["item_en"])},',
        f'\t\tAliases: []string{{{", ".join(gostr(a) for a in meta["aliases"])}}},',
        f'\t\tUpstream: {gostr(meta["upstream"])},',
        f'\t\tSite: {gostr(meta["site"])},',
        f'\t\tDM: {gostr(meta["dm"])},',
        '\t\tLoginMethods: []LoginMethod{',
    ]
    for m in MATRIX['login'].get(p, []):
        out.append(f'\t\t\t{{Method: Login{m.capitalize()}}},')
    out.append('\t\t},')
    out.append('\t\tConfigFields: []ConfigField{')
    for name, typ, required, zh, en in CONFIG[p]:
        out.append(f'\t\t\t{{Name: {gostr(name)}, Type: {gostr(typ)}, Required: {str(required).lower()},'
                   f' Description: {gostr(zh)}, DescriptionEN: {gostr(en)},'
                   f' Default: {DEFAULTS.get(name, 0)}}},')
    out.append('\t\t},')
    caps = ', '.join(gostr(c) for c in capabilities(commands, p))
    out.append(f'\t\tCapabilities: []string{{{caps}}},')
    out.append('\t\tNotes: []string{')
    for n in notes.get(p, []):
        out.append(f'\t\t\t{gostr(n)},')
    out.append('\t\t},')
    out.append(f'\t\tStage: Stage{STAGES[p]},')
    out.append('\t}')
    out.append('}')
    out.append('')
    (OUT / f'platform_{p.replace("-", "_")}.go').write_text('\n'.join(out), encoding='utf-8')


def write_commands(platforms, commands):
    out = ['// Code generated by scripts/gen_platformspec.py; DO NOT EDIT.', '', 'package platformspec', '',
           'func init() {']
    for c in commands:
        out += [
            '\tregisterCommand(&Command{',
            f'\t\tResource: {gostr(c["resource"])}, Action: {gostr(c["action"])}, '
            f'Section: {gostr(c["section"])},',
            '\t\tSupport: map[string]Support{',
        ]
        for p in platforms:
            state = c['platforms'][p]
            if state != 'unsupported':
                out.append(f'\t\t\t{gostr(p)}: Support{state.capitalize()},')
        out += ['\t\t},', '\t})']
    out += ['}', '']
    (OUT / 'commands.go').write_text('\n'.join(out), encoding='utf-8')


def main():
    platforms = MATRIX['platforms']
    commands = [dict(c) for c in MATRIX['commands']]

    notes = {}
    for n in MATRIX['notes']:
        notes.setdefault(n['platform'], []).append(f"{n['command']}: {n['text']}")

    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / 'doc.go').write_text(
        '// Package platformspec 保存从 catbus 迁移过来的平台注册表。\n'
        '//\n'
        '// catbus（MIT）把命令解析、`platforms` 命令和 docs/capabilities.md 能力矩阵都从同一份\n'
        '// 平台注册表生成。这里保留同样的单一数据源：加平台、改能力都只动 platformspec，\n'
        '// 命令树、能力矩阵和 manifest 都从它派生。\n'
        '//\n'
        '// 源仓库：https://github.com/cv-cat/catbus （MIT，见 NOTICE-catbus.md）。\n'
        '// 重新生成见 scripts/gen_platformspec.py。\n'
        'package platformspec\n', encoding='utf-8')

    for ordinal, p in enumerate(platforms):
        write_platform(p, commands, notes, ordinal)
    write_commands(platforms, commands)
    fmt()


def fmt():
    gofmt = shutil.which('gofmt')
    if not gofmt:
        return
    for f in OUT.glob('*.go'):
        subprocess.run([gofmt, '-w', str(f)], check=True)


if __name__ == '__main__':
    main()

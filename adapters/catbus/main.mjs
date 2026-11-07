#!/usr/bin/env node
/**
 * 猫巴士独立式接入器（E-SP-Line2 runtime_type: standalone）
 * ==========================================================
 *
 * 这个进程和 xianyu/taobao 的桥式接入器不一样：它**不回连** /ws/adapter。
 * 后端拉起它以后，把命令写一行 JSON 到 stdin，它执行命令、把一行 JSON 信封
 * 写到 stdout。谁也不需要连谁，进程自己就是接入器。
 *
 * 线协议（行分隔，两向都只有 JSON）：
 *
 *   in : {"id","command_type","payload":{...}}     // 后端 -> 本进程
 *   out: {"id","ok":true,"data":{...},"error":null} // 本进程 -> 后端
 *
 * 命令词表与信封字段照 catbus 源码（AGENTS.md 4.1 / 6.1）：后端送来的
 * command_type 是桥命令（send_text / send_image），这里翻译成 catbus 的命令
 * 行 `catbus <platform> msg send ...`，取回 catbus 自己的信封，原样放进
 * data 或 error，不做二次发明。对不上的地方在完成评论里写清楚。
 *
 * 日志走 stderr，格式与现有容器相同：
 *   `时间 | 级别 | [tag] 消息` + ESPL 艺术字横幅。
 * stdout 只走信封，所以日志不会污染命令通道。
 */

import { spawn } from 'node:child_process'
import { createInterface } from 'node:readline'
import { readFileSync } from 'node:fs'
import process from 'node:process'

// --------------------------------------------------------------- banner ----
// 与 adapters/xianyu/main.py:249 的 BANNER、后端 adapterBanner 完全一致。
const BANNER = String.raw`
  ███████╗   ███████╗██████╗     ██╗     ██╗███╗   ██╗███████╗██████╗
  ██╔════╝   ██╔════╝██╔══██╗    ██║     ██║████╗  ██║██╔════╝╚════██╗
  █████╗     ███████╗██████╔╝    ██║     ██║██╔██╗ ██║█████╗    ▄███╔╝
  ██╔══╝     ╚════██║██╔═══╝     ██║     ██║██║╚██╗██║██╔══╝  ▄▀══╝
  ███████╗   ███████║██║         ███████╗██║██║ ╚████║███████╗███████╗
  ╚══════╝   ╚══════╝╚═╝         ╚══════╝╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝

  Power By LangBot-community-team

  --------------------------------------------------------------------
`

// ---------------------------------------------------------------- logger ---
// loguru 的等价物：单行、定宽级别、[tag] 前缀，全部走 stderr。
// colorize=False 的等价物：不带 ANSI，日志进文件后与桥式接入器逐字同构。
const LEVELS = { debug: 'DEBUG', info: 'INFO', warning: 'WARNING', error: 'ERROR' }

function stamp() {
  const d = new Date()
  const p = (n, w = 2) => String(n).padStart(w, '0')
  return (
    `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ` +
    `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
  )
}

function log(level, tag, message) {
  const name = LEVELS[level] ?? 'INFO'
  process.stderr.write(`${stamp()} | ${name.padEnd(8)} | [${tag}] ${message}\n`)
}

const logInfo = (tag, m) => log('info', tag, m)
const logWarn = (tag, m) => log('warning', tag, m)
const logError = (tag, m) => log('error', tag, m)

// ---------------------------------------------------------------- config ---
// --config 指向运行器写下的 <instance_dir>/config.runtime.json（0600）。
const argv = process.argv.slice(2)
const flag = (name, fallback = '') => {
  const i = argv.indexOf(name)
  return i >= 0 && argv[i + 1] ? argv[i + 1] : fallback
}

const SELF_CHECK = argv.includes('--self-check')
const INSTANCE_ID = flag('--instance-id', process.env.ESP_INSTANCE_ID || 'unknown')
const BACKEND = flag('--backend', process.env.ESP_BACKEND_URL || '')
const PLATFORM = 'xianyu' // catbus 规范 id；别名 goofish/xy 由 catbus 自己归一

function loadConfig() {
  const path = flag('--config')
  if (!path) return {}
  try {
    return JSON.parse(readFileSync(path, 'utf8'))
  } catch (err) {
    logWarn('config', `读取实例配置失败(${path})，按空配置继续: ${err.message}`)
    return {}
  }
}

const config = loadConfig()
const ENDPOINT = String(config.default_endpoint || 'web').trim() || 'web'
const ACCOUNT = String(config.account || '').trim()
const CATBUS_HOME = String(config.catbus_home || '').trim()
const CATBUS_BIN_ENV = String(process.env.CATBUS_BIN || config.catbus_bin || '').trim()

// --------------------------------------------------------------- catbus ----
/**
 * 解析 catbus 可执行文件。实例配置 > 环境变量 > PATH 里的 catbus >
 * `npx --yes catbus-cli`（全局没装时的兜底）。
 *
 * 返回 {cmd, baseArgs}，调用方把命令词拼在 baseArgs 后面。
 */
function catbusCommand() {
  if (CATBUS_BIN_ENV) {
    // 允许带参数，例如 "node /path/to/catbus/dist/cli/main.js"。
    const [cmd, ...baseArgs] = CATBUS_BIN_ENV.split(/\s+/).filter(Boolean)
    return { cmd, baseArgs }
  }
  return { cmd: 'npx', baseArgs: ['--yes', 'catbus-cli'] }
}

/**
 * 执行一次 catbus 命令，解析它 stdout 上唯一的 JSON 信封。
 *
 * catbus 只在 stdout 写一个信封，日志在 stderr；stderr 原样转发到我们的
 * stderr（带 [catbus] 前缀），这样错误上下文不会丢。
 * 命令行的账号/端/代理都用全局选项，词表里不再重复。
 */
function runCatbus(words, { global = false } = {}) {
  return new Promise((resolve) => {
    const { cmd, baseArgs } = catbusCommand()
    const args = [...baseArgs, ...words]
    // 全局命令（version/doctor/platforms）不接受平台选项。
    if (!global) {
      if (ENDPOINT) args.push('-e', ENDPOINT)
      if (ACCOUNT) args.push('-a', ACCOUNT)
    }

    // catbus 自己把 NO_PROXY 设成 `*`（AGENTS 5.5：只认显式代理），所以运行器
    // 注入的 HTTP(S)_PROXY 对它无效，实例代理必须翻译成 --proxy 传进去。
    if (config.proxy && String(config.proxy) !== 'none') args.push('--proxy', String(config.proxy))

    const env = { ...process.env }
    if (CATBUS_HOME) env.CATBUS_HOME = CATBUS_HOME

    logInfo('catbus', `$ ${cmd} ${args.join(' ')}`)
    const started = Date.now()
    let child
    try {
      child = spawn(cmd, args, { env, stdio: ['ignore', 'pipe', 'pipe'] })
    } catch (err) {
      logError('catbus', `拉起失败: ${err.message}`)
      resolve({ exitCode: -1, stdout: '', stderr: err.message })
      return
    }

    let stdout = ''
    let stderr = ''
    child.stdout.setEncoding('utf8')
    child.stderr.setEncoding('utf8')
    child.stdout.on('data', (c) => (stdout += c))
    child.stderr.on('data', (c) => {
      stderr += c
      for (const line of String(c).split('\n')) {
        if (line.trim()) logWarn('catbus', line.trim())
      }
    })
    child.on('error', (err) => {
      logError('catbus', `命令进程出错: ${err.message}`)
      resolve({ exitCode: -1, stdout, stderr: stderr + err.message })
    })
    child.on('close', (code) => {
      const ms = Date.now() - started
      if (code === 0) logInfo('catbus', `完成 exit=0 用时 ${ms}ms`)
      else logWarn('catbus', `非零退出 exit=${code} 用时 ${ms}ms`)
      resolve({ exitCode: code ?? -1, stdout, stderr })
    })
  })
}

/** 从 catbus stdout 里取信封；取不到时合成一个 ERROR 信封。 */
function parseEnvelope(stdout, exitCode, stderr) {
  const lines = stdout.split('\n').map((l) => l.trim()).filter(Boolean)
  for (let i = lines.length - 1; i >= 0; i--) {
    if (!lines[i].startsWith('{')) continue
    try {
      return JSON.parse(lines[i])
    } catch {
      /* 继续往前找 */
    }
  }
  return {
    ok: false,
    data: null,
    page: null,
    error: {
      code: 'ERROR',
      message: `catbus 没有输出可解析的 JSON 信封 (exit=${exitCode})`,
      hint: null,
      detail: stderr.trim().slice(0, 2000) || null,
    },
  }
}

// ----------------------------------------------------- command mapping ----
/**
 * 桥命令 -> catbus 命令词。只翻译有对应物的两种写操作：
 *   send_text  -> catbus xianyu msg send <text> --to <toid> [--item <item>]
 *   send_image -> catbus xianyu msg send --image <url> --to <toid> [--item <item>]
 *
 * catbus 的 msg send 收会话/收信人/商品三个选项，桥 payload 的 cid 是会话
 * id、toid 是收信人，能对上就直接给；对不上时报 USAGE 而不是瞎猜。
 */
function toWords(commandType, payload) {
  const to = String(payload.toid || payload.target_id || '').trim()
  const item = String(payload.item_id || payload.item || '').trim()

  if (commandType === 'send_text') {
    const text = String(payload.text ?? payload.message_content ?? '')
    if (!text.trim()) return { error: 'USAGE: send_text 缺少 text' }
    const words = ['msg', 'send', text]
    if (item) words.push('--item', item)
    if (to) words.push('--to', to)
    return { words }
  }

  if (commandType === 'send_image') {
    const url = String(payload.image_url || payload.url || '').trim()
    if (!url) return { error: 'USAGE: send_image 缺少 image_url' }
    const words = ['msg', 'send', '--image', url]
    if (item) words.push('--item', item)
    if (to) words.push('--to', to)
    return { words }
  }

  return { error: `UNSUPPORTED: 独立式接入器尚未映射命令 ${commandType}` }
}

/** 处理一行命令，写回一行信封。 */
async function handleCommand(line) {
  let msg
  try {
    msg = JSON.parse(line)
  } catch (err) {
    const id = String(Date.now())
    logError('cmd', `无法解析命令 JSON: ${err.message}`)
    writeEnvelope({ id, ok: false, data: null, error: { code: 'USAGE', message: err.message, hint: null, detail: null } })
    return
  }

  const id = String(msg.id ?? '')
  const commandType = String(msg.command_type || msg.type || '')
  const payload = msg.payload && typeof msg.payload === 'object' ? msg.payload : {}

  logInfo('cmd', `收到 ${commandType || '(空)'} id=${id} cid=${payload.cid ?? '-'} toid=${payload.toid ?? '-'}`)
  if (payload.text) logInfo('cmd', `正文: ${String(payload.text).slice(0, 200)}`)
  if (payload.image_url) logInfo('cmd', `图片: ${payload.image_url}`)

  // stdin 上只有两种东西：后端命令和 catbus 的原始命令（command_type 为
  // "cli"，args 是已经拼好的 catbus 词表）。后者给「读一条商品」这类不经过
  // 桥命令写路径的调用，前端/API 可以直接把 catbus 命令词交给实例。
  if (commandType === 'cli' || commandType === 'catbus') {
    const words = Array.isArray(payload.args) ? payload.args.map(String) : []
    if (words.length === 0) {
      logError('cli', 'cli 命令缺少 payload.args')
      writeEnvelope({ id, ok: false, data: null, error: { code: 'USAGE', message: 'cli 命令缺少 payload.args', hint: null, detail: null } })
      return
    }
    let { exitCode, stdout, stderr } = await runCatbus(words)
    if (exitCode === 6) {
      logWarn('retry', '网络类失败，退避 2s 后重试一次')
      await new Promise((r) => setTimeout(r, 2000))
      logInfo('retry', '重试中')
      ;({ exitCode, stdout, stderr } = await runCatbus(words))
    }
    const env = parseEnvelope(stdout, exitCode, stderr)
    if (env.ok) {
      markConnected()
      logInfo('ack', `cli 成功 id=${id} exit=${exitCode}`)
    }
    else logError('ack', `cli 失败 id=${id} exit=${exitCode} code=${env.error?.code ?? 'ERROR'} ${env.error?.message ?? ''}`)
    writeEnvelope({ id, ok: Boolean(env.ok), data: env.data ?? null, page: env.page ?? null, error: env.error ?? null })
    return
  }

  const mapped = toWords(commandType, payload)
  if (mapped.error) {
    logError('cmd', mapped.error)
    writeEnvelope({ id, ok: false, data: null, error: { code: mapped.error.split(':')[0], message: mapped.error, hint: null, detail: null } })
    return
  }

  // catbus 的第一个位置参数是平台 id，接在全局选项之前。
  let { exitCode, stdout, stderr } = await runCatbus([PLATFORM, ...mapped.words])
  // 6 是 catbus 的 NETWORK（连接 / 代理）。网络抖一下不该让调用方直接失败：
  // 退避 2 秒重试一次，并把重连写进日志。
  if (exitCode === 6) {
    logWarn('retry', '网络类失败，退避 2s 后重试一次')
    await new Promise((r) => setTimeout(r, 2000))
    logInfo('retry', '重试中')
    ;({ exitCode, stdout, stderr } = await runCatbus([PLATFORM, ...mapped.words]))
  }
  const env = parseEnvelope(stdout, exitCode, stderr)
  if (env.ok) {
    markConnected()
    logInfo('ack', `发送成功 id=${id} exit=${exitCode}`)
  } else {
    logError('ack', `发送失败 id=${id} exit=${exitCode} code=${env.error?.code ?? 'ERROR'} ${env.error?.message ?? ''}`)
  }
  writeEnvelope({ id, ok: Boolean(env.ok), data: env.data ?? null, page: env.page ?? null, error: env.error ?? null })
}

// 独立式没有长连接，"连上"指第一次成功跑通 catbus（CLI 可用、平台可达）。
// 只在第一次记一行，避免每条命令刷屏。
let connected = false
function markConnected() {
  if (connected) return
  connected = true
  logInfo('conn', `已连通 catbus（${PLATFORM}，endpoint=${ENDPOINT}）`)
}

/** 一行一个信封，写 stdout 并立即 flush。 */
function writeEnvelope(env) {
  process.stdout.write(`${JSON.stringify(env)}\n`)
}

// ------------------------------------------------------- tool catalog -----
/**
 * 启动时上报可供外部框架调用的工具（inactive fields），与桥式接入器在 WebSocket
 * 上发的 tool_catalog 帧同构。独立式没有 socket，所以走 stdout：运行器把它交给
 * 网关的同一个注册函数。
 *
 * 这里只登记 catbus 词表里确实存在的读取类命令，参数名与词表一致。
 */
const TOOL_CATALOG = {
  type: 'tool_catalog',
  payload: {
    tools: [
      { name: 'xianyu_item_get', semantic_id: 'item_get', category: 'query_only', platform: PLATFORM,
        description: '闲鱼商品详情', parameters: [{ name: 'item', type: 'string', required: true }], enabled: true },
      { name: 'xianyu_item_search', semantic_id: 'item_search', category: 'query_only', platform: PLATFORM,
        description: '闲鱼商品搜索', parameters: [{ name: 'keyword', type: 'string', required: true }], enabled: true },
      { name: 'xianyu_user_get', semantic_id: 'user_get', category: 'query_only', platform: PLATFORM,
        description: '闲鱼用户资料', parameters: [{ name: 'user', type: 'string', required: true }], enabled: true },
      { name: 'xianyu_msg_list', semantic_id: 'msg_list', category: 'query_only', platform: PLATFORM,
        description: '闲鱼会话列表', parameters: [], enabled: true },
      { name: 'xianyu_msg_history', semantic_id: 'msg_history', category: 'query_only', platform: PLATFORM,
        description: '会话消息记录', parameters: [{ name: 'conversation', type: 'string', required: true }], enabled: true },
      { name: 'xianyu_doctor', semantic_id: 'doctor', category: 'query_only', platform: PLATFORM,
        description: 'catbus 环境自检', parameters: [], enabled: true },
    ],
  },
}

// ------------------------------------------------------------ self-check ---
/**
 * `--self-check`：实例创建时由依赖安装器调用（不是运行时路径）。
 * 检查 Node、catbus 可执行文件、配置文件，结果打成一封 JSON 信封。
 * 有任一必需项缺失时退出码 1，让实例停在 error 状态并显示原因。
 */
async function selfCheck() {
  const checks = []
  const add = (name, ok, message, required = true) => checks.push({ name, ok, message, required })

  const nodeMajor = Number(process.versions.node.split('.')[0])
  add('node', nodeMajor >= 22, `Node ${process.versions.node}（catbus 要求 ^22.22.2 || ^24.15.0 || >=26）`)

  const { cmd, baseArgs } = catbusCommand()
  const probe = await runCatbus(['version'], { global: true })
  let cliOk = false
  let cliMsg = `${cmd} 不可用（exit=${probe.exitCode}）`
  if (probe.exitCode === 0) {
    const env = parseEnvelope(probe.stdout, probe.exitCode, probe.stderr)
    cliOk = Boolean(env.ok)
    cliMsg = cliOk ? `${cmd} ${env.data?.version ?? ''} 可用` : `catbus 版本检查失败: ${env.error?.message ?? ''}`
  } else if (probe.stderr) {
    cliMsg += `: ${probe.stderr.trim().split('\n')[0]}`
  }
  add('catbus', cliOk, cliMsg)

  add('config', true, `实例配置: account=${ACCOUNT || '(当前)'} endpoint=${ENDPOINT} home=${CATBUS_HOME || '(默认)'}`, false)
  add('platform', true, `平台: ${PLATFORM}`, false)

  for (const c of checks) {
    if (c.ok) logInfo('selfcheck', `${c.name}: ${c.message}`)
    else if (c.required) logError('selfcheck', `${c.name}: ${c.message}`)
    else logWarn('selfcheck', `${c.name}: ${c.message}`)
  }

  const failed = checks.filter((c) => c.required && !c.ok)
  writeEnvelope({
    id: 'self-check',
    ok: failed.length === 0,
    data: { instance_id: INSTANCE_ID, checks },
    page: null,
    error: failed.length
      ? {
          code: 'ERROR',
          message: `自检未通过: ${failed.map((c) => c.name).join(', ')}`,
          hint: '安装 Node 22+ 与 catbus（npm i -g catbus-cli），或改实例配置里的 catbus 可执行文件路径',
          detail: null,
        }
      : null,
  })
  return failed.length === 0 ? 0 : 1
}

// ------------------------------------------------------------------ main ---
async function main() {
  process.stderr.write(BANNER)
  logInfo('start', `猫巴士独立式接入器启动 instance=${INSTANCE_ID} platform=${PLATFORM} endpoint=${ENDPOINT} backend=${BACKEND || '-'}`)
  logInfo('start', `cfg account=${ACCOUNT || '(当前)'} catbus=${CATBUS_BIN_ENV || '(PATH/npx)'} home=${CATBUS_HOME || '(默认)'}`)
  logInfo('start', '模式: 独立式（命令经 stdin，一行一个 JSON；结果走 stdout 信封，不回连 WebSocket）')

  if (SELF_CHECK) {
    const code = await selfCheck()
    process.exit(code)
  }

  // 先上报工具目录，再进入命令循环。
  process.stdout.write(`${JSON.stringify(TOOL_CATALOG)}\n`)
  logInfo('start', `已上报 ${TOOL_CATALOG.payload.tools.length} 个 inactive-field 工具`)

  logInfo('start', '等待命令 (stdin)…')
  const rl = createInterface({ input: process.stdin, crlfDelay: Infinity })
  let inflight = 0

  rl.on('line', async (line) => {
    const trimmed = line.trim()
    if (!trimmed) return
    inflight++
    try {
      await handleCommand(trimmed)
    } catch (err) {
      logError('cmd', `命令处理异常: ${err?.stack || err}`)
    } finally {
      inflight--
    }
  })

  rl.on('close', () => {
    logInfo('stop', `stdin 关闭，等待 ${inflight} 条在途命令后退出`)
    const wait = () => (inflight > 0 ? setTimeout(wait, 50) : process.exit(0))
    wait()
  })

  // 后端发 SIGTERM 是正常停止，不算错误。
  for (const sig of ['SIGTERM', 'SIGINT']) {
    process.on(sig, () => {
      logInfo('stop', `收到 ${sig}，退出`)
      process.exit(0)
    })
  }
}

main().catch((err) => {
  logError('fatal', String(err?.stack || err))
  process.exit(1)
})

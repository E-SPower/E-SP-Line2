#!/usr/bin/env node
// Locale drift guard.
//
// `web/DOCS.md` claimed a test compared the two language packs; there was
// none, so a key added to one language and forgotten in the other shipped as a
// raw `logs.title` string in the UI. This is that test, written without a
// runner so it needs no new dependency: `npm run test:locales`.
//
// Two checks:
//   1. both packs have exactly the same key set, and neither is empty;
//   2. every `t('...')` literal in the source resolves in at least one pack.
//
// Check 2 covers static call sites only. `t(variable)` and keys reached via a
// computed prefix are out of reach; the parity check still keeps the packs
// aligned with each other, which is the drift that actually bites.

import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const localesDir = join(root, 'src', 'locales')
const locales = readdirSync(localesDir).filter((f) => f.endsWith('.json'))
const reference = 'en-US.json'

const flatten = (obj, prefix = '', out = new Set()) => {
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object' && !Array.isArray(v)) flatten(v, key, out)
    else out.add(key)
  }
  return out
}

const load = (file) => JSON.parse(readFileSync(join(localesDir, file), 'utf8'))

const problems = []
const packs = new Map()
for (const file of locales) {
  const flat = flatten(load(file))
  packs.set(file, flat)
  if (flat.size === 0) problems.push(`${file}: empty`)
}

// 1. parity
const refKeys = packs.get(reference)
if (!refKeys) {
  problems.push(`missing reference locale ${reference}`)
} else {
  for (const [file, keys] of packs) {
    if (file === reference) continue
    const missing = [...refKeys].filter((k) => !keys.has(k))
    const extra = [...keys].filter((k) => !refKeys.has(k))
    for (const k of missing) problems.push(`${file}: missing key "${k}" (present in ${reference})`)
    for (const k of extra) problems.push(`${file}: key "${k}" absent from ${reference}`)
  }
}

// 2. static call sites resolve somewhere
const walk = (dir, out = []) => {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    if (statSync(full).isDirectory()) walk(full, out)
    else if (/\.tsx?$/.test(entry) && !/\.test\./.test(entry)) out.push(full)
  }
  return out
}

const USE = /\bt\(\s*'([A-Za-z][\w.]*)'/g
for (const file of walk(join(root, 'src'))) {
  const src = readFileSync(file, 'utf8')
  const rel = file.slice(root.length + 1)
  for (const [, key] of src.matchAll(USE)) {
    if (![...packs.values()].some((keys) => keys.has(key))) {
      problems.push(`${rel}: t('${key}') does not resolve in any locale`)
    }
  }
}

if (problems.length) {
  console.error(`locale check failed (${problems.length}):`)
  for (const p of problems) console.error(`  ${p}`)
  process.exit(1)
}
console.log(`locale check ok: ${locales.length} packs, ${refKeys.size} keys in agreement`)

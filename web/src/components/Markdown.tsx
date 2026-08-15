import { Fragment, ReactNode } from 'react'

/**
 * A lightweight, dependency-free markdown renderer.
 *
 * Supports the subset of markdown used across the project documentation:
 * - ATX headings (# .. ######)
 * - Fenced code blocks (```) with language hint
 * - Inline code (`code`)
 * - Unordered/ordered lists (including nested)
 * - Tables (GitHub-style pipes)
 * - Blockquotes (>)
 * - Bold (**text** / __text__), italic (*text* / _text_)
 * - Inline links [text](url) and bare URLs
 * - Thematic breaks (---)
 * - Paragraphs and hard breaks
 *
 * All text is escaped before rendering (React escapes by default when passed
 * as children), so arbitrary HTML inside the documents cannot be injected.
 */

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&')
    .replace(/</g, '<')
    .replace(/>/g, '>')
}

/** Convert a raw heading string into an anchor id (mirrors GitHub slugify). */
function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\u4e00-\u9fa5\s-]/g, '')
    .replace(/\s+/g, '-')
}

/** Parse inline markdown (code, bold, italic, links) into React nodes. */
function parseInline(text: string): ReactNode[] {
  const nodes: ReactNode[] = []
  let remaining = text
  let key = 0

  // Regex alternatives in priority order:
  // 1. inline code  2. image  3. link  4. bold  5. italic  6. break  7. plain
  const pattern =
    /(`[^`]+`)|(!\[[^\]]*\]\([^)]*\))|(\[[^\]]*\]\([^)]*\))|(\*\*[^*]+\*\*|__[^_]+__)|(\*[^*]+\*|_[^_]+_)|(\n)/g

  let match: RegExpExecArray | null
  let lastIndex = 0

  while ((match = pattern.exec(remaining)) !== null) {
    if (match.index > lastIndex) {
      nodes.push(<Fragment key={key++}>{escapeHtml(remaining.slice(lastIndex, match.index))}</Fragment>)
    }

    const [full] = match
    const [, code, image, link, bold, italic, brk] = match

    if (code) {
      nodes.push(
        <code
          key={key++}
          className="px-1.5 py-0.5 rounded bg-gray-100 text-pink-600 text-sm font-mono"
        >
          {escapeHtml(code.slice(1, -1))}
        </code>
      )
    } else if (image) {
      const inner = image.slice(2, -1) // strip ![ and )
      const sep = inner.indexOf('](')
      if (sep > -1) {
        const alt = inner.slice(0, sep)
        const src = inner.slice(sep + 2)
        nodes.push(
          <img
            key={key++}
            src={src}
            alt={alt}
            className="max-w-full h-auto rounded-lg my-2"
            loading="lazy"
          />
        )
      }
    } else if (link) {
      const inner = link.slice(1, -1) // strip [ and )
      const sep = inner.indexOf('](')
      if (sep > -1) {
        const label = inner.slice(0, sep)
        const href = inner.slice(sep + 2)
        nodes.push(
          <a
            key={key++}
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            className="text-blue-600 hover:text-blue-800 hover:underline"
          >
            {parseInline(label)}
          </a>
        )
      }
    } else if (bold) {
      nodes.push(
        <strong key={key++} className="font-semibold">
          {parseInline(bold.slice(2, -2))}
        </strong>
      )
    } else if (italic) {
      nodes.push(<em key={key++}>{parseInline(italic.slice(1, -1))}</em>)
    } else if (brk) {
      nodes.push(<br key={key++} />)
    }

    lastIndex = match.index + full.length
  }

  if (lastIndex < remaining.length) {
    nodes.push(<Fragment key={key++}>{escapeHtml(remaining.slice(lastIndex))}</Fragment>)
  }

  return nodes
}

/** Split fenced code blocks out of a raw markdown source. */
function tokenize(source: string): Array<{ type: 'code' | 'text'; lang?: string; content: string }> {
  const tokens: Array<{ type: 'code' | 'text'; lang?: string; content: string }> = []
  const lines = source.split('\n')
  let i = 0

  while (i < lines.length) {
    const line = lines[i]
    const fence = line.match(/^```(\w*)\s*$/)
    if (fence) {
      const lang = fence[1]
      const codeLines: string[] = []
      i++
      while (i < lines.length && !/^```\s*$/.test(lines[i])) {
        codeLines.push(lines[i])
        i++
      }
      i++ // skip closing fence
      tokens.push({ type: 'code', lang, content: codeLines.join('\n') })
    } else {
      const textLines: string[] = []
      while (i < lines.length && !/^```/.test(lines[i])) {
        textLines.push(lines[i])
        i++
      }
      tokens.push({ type: 'text', content: textLines.join('\n') })
    }
  }

  return tokens
}

/** Render the text portion of a document (headings, lists, tables, etc.). */
function renderTextBlock(text: string): ReactNode[] {
  const blocks: ReactNode[] = []
  const lines = text.split('\n')
  let i = 0
  let key = 0

  while (i < lines.length) {
    const line = lines[i]

    // Blank line -> paragraph break
    if (!line.trim()) {
      i++
      continue
    }

    // Heading
    const heading = line.match(/^(#{1,6})\s+(.+)$/)
    if (heading) {
      const level = heading[1].length
      const content = heading[2].trim()
      const Tag = (`h${level}`) as 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6'
      blocks.push(
        <Tag
          key={key++}
          id={slugify(content)}
          className={
            level === 1
              ? 'text-2xl font-bold text-gray-900 mt-8 mb-3 pb-2 border-b border-gray-200'
              : level === 2
                ? 'text-xl font-semibold text-gray-900 mt-6 mb-2'
                : 'text-lg font-semibold text-gray-800 mt-5 mb-2'
          }
        >
          {parseInline(content)}
        </Tag>
      )
      i++
      continue
    }

    // Thematic break
    if (/^\s*(-{3,}|\*{3,}|_{3,})\s*$/.test(line)) {
      blocks.push(<hr key={key++} className="my-6 border-gray-200" />)
      i++
      continue
    }

    // Blockquote
    if (line.startsWith('>')) {
      const quoteLines: string[] = []
      while (i < lines.length && lines[i].startsWith('>')) {
        quoteLines.push(lines[i].replace(/^>\s?/, ''))
        i++
      }
      blocks.push(
        <blockquote
          key={key++}
          className="border-l-4 border-blue-200 pl-4 py-1 my-3 text-gray-600 italic"
        >
          {renderTextBlock(quoteLines.join('\n'))}
        </blockquote>
      )
      continue
    }

    // Table: gather consecutive lines containing pipes; require a separator row.
    if (line.includes('|') && i + 1 < lines.length && /^\s*\|?[\s:|-]+\|?\s*$/.test(lines[i + 1])) {
      const headerCells = line
        .replace(/^\||\|$/g, '')
        .split('|')
        .map((c) => c.trim())
      i += 2 // skip header + separator
      const rows: string[][] = []
      while (i < lines.length && lines[i].includes('|')) {
        rows.push(
          lines[i]
            .replace(/^\||\|$/g, '')
            .split('|')
            .map((c) => c.trim())
        )
        i++
      }
      blocks.push(
        <div key={key++} className="overflow-x-auto my-4">
          <table className="min-w-full text-sm border-collapse">
            <thead>
              <tr className="bg-gray-50">
                {headerCells.map((cell, idx) => (
                  <th
                    key={idx}
                    className="border border-gray-200 px-3 py-2 text-left font-semibold text-gray-700"
                  >
                    {parseInline(cell)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, rowIdx) => (
                <tr key={rowIdx} className={rowIdx % 2 ? 'bg-gray-50/50' : ''}>
                  {headerCells.map((_, colIdx) => (
                    <td key={colIdx} className="border border-gray-200 px-3 py-2 text-gray-600">
                      {parseInline(row[colIdx] ?? '')}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )
      continue
    }

    // List: unordered (-, *, +) or ordered (1.)
    if (/^\s*([-*+]|\d+\.)\s+/.test(line)) {
      const ordered = /^\s*\d+\.\s+/.test(line)
      const items: { depth: number; content: string }[] = []
      while (i < lines.length) {
        const l = lines[i]
        const listMatch = l.match(/^(\s*)([-*+]|\d+\.)\s+(.*)$/)
        if (!listMatch) {
          // continuation line of the previous item
          if (items.length && l.trim() && !/^\s*$/.test(l)) {
            items[items.length - 1].content += ' ' + l.trim()
            i++
            continue
          }
          break
        }
        const depth = Math.floor(listMatch[1].replace(/\t/g, '  ').length / 2)
        items.push({ depth, content: listMatch[3].trim() })
        i++
      }

      // Build a flat list of list items; nesting is rendered by indentation.
      const ListTag = ordered ? 'ol' : 'ul'
      const listItems = items.map((item, idx) => (
        <li
          key={idx}
          className="py-0.5"
          style={{ marginLeft: `${item.depth * 1.5}rem` }}
        >
          {parseInline(item.content)}
        </li>
      ))
      blocks.push(
        <ListTag
          key={key++}
          className={ordered ? 'list-decimal pl-6 my-3 text-gray-700' : 'list-disc pl-6 my-3 text-gray-700'}
        >
          {listItems}
        </ListTag>
      )
      continue
    }

    // Plain paragraph (gather until blank line or another block start)
    const paraLines: string[] = []
    while (
      i < lines.length &&
      lines[i].trim() &&
      !/^(#{1,6})\s/.test(lines[i]) &&
      !/^```/.test(lines[i]) &&
      !lines[i].startsWith('>') &&
      !/^\s*([-*+]|\d+\.)\s+/.test(lines[i])
    ) {
      paraLines.push(lines[i])
      i++
    }
    blocks.push(
      <p key={key++} className="my-3 text-gray-700 leading-relaxed">
        {parseInline(paraLines.join(' '))}
      </p>
    )
  }

  return blocks
}

export default function Markdown({ source }: { source: string }) {
  const tokens = tokenize(source)
  let key = 0

  return (
    <div className="prose max-w-none">
      {tokens.map((token) =>
        token.type === 'code' ? (
          <pre
            key={key++}
            className="my-4 p-4 rounded-lg bg-gray-900 text-gray-100 overflow-x-auto text-sm"
          >
            <code className="font-mono">{escapeHtml(token.content)}</code>
          </pre>
        ) : (
          <Fragment key={key++}>{renderTextBlock(token.content)}</Fragment>
        )
      )}
    </div>
  )
}

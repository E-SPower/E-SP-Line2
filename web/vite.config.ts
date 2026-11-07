import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import fs from 'node:fs'
import type { IncomingMessage, ServerResponse } from 'node:http'
import path from 'node:path'

/** 只声明 docs 中间件真正用到的那一小部分 connect 接口，避免用 `Function` 丢掉类型。 */
type DocsMiddleware = {
  use: (
    route: string,
    handler: (req: IncomingMessage, res: ServerResponse, next: (err?: unknown) => void) => void,
  ) => void
}

/**
 * 把仓库 docs/ 以 /docs 提供给浏览器：文档页直接链到这里的 Markdown 原文。
 * 找不到时回 404，不能落到 SPA 的 index.html，否则断链会静默显示成应用首页。
 */
const docsRoot = path.resolve(__dirname, '..', 'docs')

const docsPlugin = () => ({
  name: 'serve-docs',
  configureServer(server: { middlewares: DocsMiddleware }) {
    server.middlewares.use('/docs', (req: IncomingMessage, res: ServerResponse, next: () => void) => {
      const rel = decodeURIComponent((req.url ?? '/').split('?')[0])
      const file = path.resolve(docsRoot, '.' + path.posix.normalize(rel))
      // /docs 本身交给 SPA 的文档页；只有确实点到一个文件才由这里回。
      if (!file.startsWith(docsRoot) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) {
        if (file === docsRoot) return next()

        res.statusCode = 404
        res.setHeader('Content-Type', 'text/plain; charset=utf-8')
        res.end(`no such doc: ${rel}\n`)
        return
      }
      res.setHeader('Content-Type', 'text/markdown; charset=utf-8')
      fs.createReadStream(file).pipe(res)
    })
  },
})

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), docsPlugin()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
})

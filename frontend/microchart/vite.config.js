import { defineConfig } from 'vite'
import { viteSingleFile } from 'vite-plugin-singlefile'
import zlib from 'zlib'
import fs from 'fs'

function sizeReport() {
  return {
    name: 'size-report',
    closeBundle() {
      const html = fs.readFileSync('dist/index.html')
      const gz   = zlib.gzipSync(html)
      const min  = (html.length / 1024).toFixed(1) + ' KB'
      const gzs  = (gz.length  / 1024).toFixed(1) + ' KB'
      console.log(`\n  Bundle: ${min} min | ${gzs} gzip\n`)
      // Patch badges dans le HTML final
      let content = html.toString()
      content = content
        .replace('⏳ …</span>\n        <span class="badge b" id="badge-gz">⏳ …',
                 `✓ ${min}</span>\n        <span class="badge b" id="badge-gz">✓ ${gzs}`)
      fs.writeFileSync('dist/index.html', content)
    }
  }
}

export default defineConfig({
  plugins: [viteSingleFile(), sizeReport()],
  build: { minify: 'esbuild', assetsInlineLimit: Infinity }
})

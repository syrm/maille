import { defineConfig } from 'vite'
import { viteSingleFile } from 'vite-plugin-singlefile'
import { execSync } from 'child_process'
import zlib from 'zlib'
import fs from 'fs'
import path from 'path'

// Plugin qui mesure la taille du bundle après build et l'injecte dans le HTML
function bundleSizePlugin() {
  return {
    name: 'bundle-size',
    closeBundle() {
      const distDir = path.resolve('dist')
      if (!fs.existsSync(distDir)) return
      const html = fs.readdirSync(distDir).find(f => f.endsWith('.html'))
      if (!html) return
      const content = fs.readFileSync(path.join(distDir, html))
      const gz = zlib.gzipSync(content)
      const minKb = (content.length / 1024).toFixed(1) + ' KB'
      const gzKb  = (gz.length / 1024).toFixed(1) + ' KB'
      console.log(`\n  microchart bundle: ${minKb} min | ${gzKb} gzip\n`)
    }
  }
}

export default defineConfig({
  plugins: [viteSingleFile(), bundleSizePlugin()],
  define: {
    __BUNDLE_MIN__: '"?"',
    __BUNDLE_GZ__:  '"?"',
  },
  build: {
    minify: 'esbuild',
    assetsInlineLimit: Infinity,
  }
})

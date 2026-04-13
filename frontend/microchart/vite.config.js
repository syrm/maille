import { defineConfig } from 'vite'

export default defineConfig({
  build: {
    minify: 'esbuild',
    assetsInlineLimit: Infinity,
    outDir: '../../internal/web/dist',
    rollupOptions: {
      input: './src/Index.res.mjs',
      output: {
        entryFileNames: 'index.js',
        assetFileNames: 'index.js',
        chunkFileNames: 'index.js'
      }
    }
  }
})

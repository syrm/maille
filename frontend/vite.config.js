import { defineConfig } from 'vite'

export default defineConfig({
  build: {
    minify: false, // 'esbuild',
    assetsInlineLimit: Infinity,
    outDir: '../internal/web/dist',
    rollupOptions: {
      input: './lib/es6/app/src/Main.res.mjs',
      preserveEntrySignatures: 'exports-only',
      output: {
        entryFileNames: 'script.min.js',
        assetFileNames: 'script.min.js',
        chunkFileNames: 'script.min.js'
      }
    }
  }
})

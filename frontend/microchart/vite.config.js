import { defineConfig } from 'vite'

export default defineConfig({
  build: {
    minify: 'esbuild',                    // ✅ déjà là
    assetsInlineLimit: Infinity,
    outDir: '../../internal/web/dist',
    rollupOptions: {
      input: './src/index.js',
      preserveEntrySignatures: 'exports-only',  // ← le fix
      output: {
        entryFileNames: 'index.js',
        assetFileNames: 'index.js',
        chunkFileNames: 'index.js'
      }
    }
  }
})

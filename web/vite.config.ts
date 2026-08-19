import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { svelteTesting } from '@testing-library/svelte/vite'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vitest/config'

const rootDir = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [tailwindcss(), svelte(), svelteTesting()],
  resolve: {
    alias: {
      // Plan contract string; package ships as @fontsource-variable/inter
      '@fontsource/inter/variable.css': path.resolve(
        rootDir,
        'node_modules/@fontsource-variable/inter/index.css',
      ),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    strictPort: true,
    hmr: { protocol: 'ws', host: 'localhost', clientPort: 8080, path: '/@vite-hmr' },
  },
  test: { environment: 'jsdom', setupFiles: ['./src/test/setup.ts'] },
})

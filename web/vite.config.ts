import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [tailwindcss(), svelte()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    strictPort: true,
    hmr: { protocol: 'ws', host: 'localhost', clientPort: 8080, path: '/@vite-hmr' },
  },
  test: { environment: 'jsdom', setupFiles: ['./src/test/setup.ts'] },
})

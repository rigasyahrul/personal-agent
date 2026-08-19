// web/tailwind.config.ts
import type { Config } from 'tailwindcss';
export default {
  content: ['./src/**/*.{html,js,svelte,ts}', './src/app.html', './index.html'],
  theme: {
    extend: {
      fontFamily: { sans: ['Inter Variable', 'Inter', 'system-ui', 'sans-serif'] },
      colors: { accent: { DEFAULT: '#2563eb', foreground: '#ffffff' } },
    },
  },
  plugins: [],
} satisfies Config;

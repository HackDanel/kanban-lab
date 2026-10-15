import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

const backend = process.env.VITE_BACKEND_URL ?? 'http://localhost:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api/v1/ws': { target: backend, ws: true },
      '/api': backend,
      '/docs': backend,
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    // формы с userEvent медленные при сборе покрытия
    testTimeout: 15_000,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/main.tsx', 'src/test/**', 'src/**/*.d.ts', 'src/**/*.test.{ts,tsx}'],
      thresholds: { lines: 80, branches: 70 },
    },
  },
})

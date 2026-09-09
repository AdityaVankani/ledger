import { defineConfig } from 'vite'

export default defineConfig({
  server: {
    proxy: {
      '/v1': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
    },
  },
})
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// In development (npm run dev), /api is forwarded to the local Go API.
// In Docker, nginx does this instead (see nginx.conf).
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})

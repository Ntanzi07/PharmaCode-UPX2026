import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Em desenvolvimento (npm run dev), /api é repassado para a API Go local.
// No Docker, quem faz esse papel é o nginx (ver nginx.conf).
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

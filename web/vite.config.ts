import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Dev-time reverse proxy: the browser only ever talks to this origin, so the
// backend's cookie auth works unchanged for both REST and websockets.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': {
        target: 'ws://localhost:8081',
        ws: true,
        rewrite: (path) => path.replace(/^\/ws/, ''),
        // the ws server only admits this exact Origin (src/cmd/ws/main.go)
        headers: { Origin: 'http://localhost:8080' },
      },
    },
  },
})

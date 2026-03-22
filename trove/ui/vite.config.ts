import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import fs from 'fs'
import path from 'path'

const certsDir = path.resolve(__dirname, '../infra/certs')
const hasCerts = fs.existsSync(path.join(certsDir, 'cert.pem'))
const backendUrl = hasCerts ? 'https://localhost:8080' : 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 3000,
    ...(hasCerts && {
      https: {
        cert: fs.readFileSync(path.join(certsDir, 'cert.pem')),
        key: fs.readFileSync(path.join(certsDir, 'key.pem')),
      },
    }),
    proxy: {
      '/api': {
        target: backendUrl,
        changeOrigin: true,
        secure: false,
        rewrite: (p) => p.replace(/^\/api/, ''),
      },
      '/uploads': {
        target: backendUrl,
        changeOrigin: true,
        secure: false,
      },
    },
  },
})

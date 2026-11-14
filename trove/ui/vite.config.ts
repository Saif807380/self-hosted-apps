import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'
import fs from 'fs'
import path from 'path'

const certsDir = path.resolve(__dirname, '../infra/certs')
// TROVE_TLS_CERT / TROVE_TLS_KEY swap in another certificate, e.g. the
// tailnet one: service workers (and so push) only register on a page whose
// certificate the browser trusts.
const certFile = process.env.TROVE_TLS_CERT ?? path.join(certsDir, 'cert.pem')
const keyFile = process.env.TROVE_TLS_KEY ?? path.join(certsDir, 'key.pem')
const hasCerts = fs.existsSync(certFile)
// TROVE_BACKEND_URL points the dev server at another backend, e.g. one running
// against a scratch copy of the database.
const backendUrl = process.env.TROVE_BACKEND_URL ?? (hasCerts ? 'https://localhost:8080' : 'http://localhost:8080')

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      // Our own service worker (src/sw.ts) so it can handle web push for sale
      // alerts; caching rules live there now.
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.ts',
      manifest: {
        name: 'Trove',
        short_name: 'Trove',
        description: 'Personal library & journal',
        theme_color: '#0e0f12',
        background_color: '#0e0f12',
        display: 'standalone',
        icons: [
          { src: '/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: '/icon-512.png', sizes: '512x512', type: 'image/png' },
        ],
      },
    }),
  ],
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
        cert: fs.readFileSync(certFile),
        key: fs.readFileSync(keyFile),
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

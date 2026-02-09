import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.VITE_API_BASE_URL || 'https://localhost:8443',
        changeOrigin: true,
        secure: false, // Allow self-signed certificates for local dev
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})

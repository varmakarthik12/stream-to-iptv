import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': 'http://localhost:8068',
      '/stream': 'http://localhost:8068',
      '/playlist.m3u': 'http://localhost:8068',
      '/epg.xml': 'http://localhost:8068',
      '/epg.xml.gz': 'http://localhost:8068',
      '/logos': 'http://localhost:8068',
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  }
})

import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist',
    rollupOptions: {
      // /wails/runtime is injected by the Wails binary at runtime; don't bundle it.
      external: ['/wails/runtime'],
    },
  },
})

import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte()],
  // AGREGAMOS ESTAS LÍNEAS:
  optimizeDeps: {
    esbuildOptions: {
        target: 'esnext'
    }
  },
  build: {
    target: 'esnext'
  }
})

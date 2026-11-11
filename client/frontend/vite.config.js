import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte()],
  publicDir: "public",
  optimizeDeps: {
    esbuildOptions: {
      target: "esnext",
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
      "@app": path.resolve(__dirname, "src/app"),
      "@shared": path.resolve(__dirname, "src/shared"),
      "@features": path.resolve(__dirname, "src/features"),
      "@wails": path.resolve(__dirname, "wailsjs"),
      wailsjs: path.resolve(__dirname, "wailsjs"),
    },
  },
  build: {
    target: "esnext",
  },
});

import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    port: 3100,
    proxy: {
      // Forward /api/* and /healthz to the Go backend in dev
      '/api': { target: 'http://127.0.0.1:8090', changeOrigin: false },
      '/healthz': { target: 'http://127.0.0.1:8090', changeOrigin: false }
    }
  }
});

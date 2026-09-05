import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";
import { previewApi } from "./scripts/preview-api.mjs";

export default defineConfig(({ command }) => {
  const preview = command === "serve" && process.env.FUND_PREVIEW === "1";
  return {
    plugins: [...(preview ? [previewApi()] : []), sveltekit()],
    define: { "import.meta.env.VITE_PREVIEW": JSON.stringify(preview) },
    server: {
      port: 3100,
      proxy: {
        // Forward /api/* and /healthz to the Go backend in dev
        "/api": { target: "http://127.0.0.1:8090", changeOrigin: false },
        "/healthz": { target: "http://127.0.0.1:8090", changeOrigin: false },
      },
    },
  };
});

import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

// Dev ports and the API target come from the repo-root .env (the same
// file the Go config reads), never from literals here.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, "../", "");
  const publicURL = new URL(env.KURA_PUBLIC_URL || "http://localhost:5174");
  const apiTarget = `http://${env.KURA_ADDR || "127.0.0.1:8094"}`;
  return {
    plugins: [react()],
    envDir: "../",
    server: {
      port: Number(publicURL.port) || 5174,
      strictPort: true,
      proxy: {
        "/api": apiTarget,
        "/media": apiTarget,
        "/health": apiTarget,
      },
    },
    build: { outDir: "../internal/webui/dist", emptyOutDir: true },
  };
});

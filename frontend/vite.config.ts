import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // In development the API runs on its own port; nginx does the same job in production,
    // so the app only ever talks to a same-origin /api and CORS never comes up.
    proxy: { "/api": { target: "http://localhost:8080", changeOrigin: true } },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test-setup.ts"],
    coverage: { provider: "v8", include: ["src/**/*.{ts,tsx}"] },
  },
});

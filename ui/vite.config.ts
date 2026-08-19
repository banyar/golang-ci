import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Dev-only proxy to the Go dashboard (make dashboard-run, port 8081) --
// avoids needing any CORS changes on the backend during development.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": "http://localhost:8081",
      "/swagger": "http://localhost:8081",
    },
  },
});

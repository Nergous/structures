import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Relative base so the build works from GitHub Pages project URLs and from
// any static file server.
export default defineConfig({
  base: "./",
  plugins: [react()],
});

import { defineConfig } from "vite";
import { fileURLToPath, URL } from "node:url";
import react from "@vitejs/plugin-react";
import wails from "@wailsio/runtime/plugins/vite";

// 共有フロントパッケージ @shinte/web（core/web）は npm install せず alias で解決する。
// shinte-web/ は scripts/shinte-web.mjs が go.mod の版の core からコピーする（dev / build / test の前に自動で走る）。
const shinteWeb = fileURLToPath(new URL("./shinte-web", import.meta.url));

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  resolve: {
    alias: { "@shinte/web": shinteWeb },
  },
  plugins: [react(), wails("./bindings")],
});

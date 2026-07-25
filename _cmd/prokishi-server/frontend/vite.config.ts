import { defineConfig } from "vite";
import { fileURLToPath, URL } from "node:url";
import react from "@vitejs/plugin-react";
import wails from "@wailsio/runtime/plugins/vite";

// 共有フロントパッケージ core/web (@shinte/web) はリポジトリ内の別ディレクトリ。
// npm install せずに参照するため alias で解決し、dev サーバの fs アクセスを許可する。
const shinteWeb = fileURLToPath(new URL("../../../../core/web", import.meta.url));
const repoRoot = fileURLToPath(new URL("../../../../", import.meta.url));

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
    fs: { allow: [repoRoot] },
  },
  resolve: {
    alias: { "@shinte/web": shinteWeb },
  },
  plugins: [react(), wails("./bindings")],
});

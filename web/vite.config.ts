import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// When bundled inside the Tauri desktop shell (TAURI=1, set by the release
// build), assets are served from tauri://localhost and must be referenced
// with relative paths. The Cloudflare Pages deployment keeps the absolute
// "/" base so routes like /loot resolve correctly. Same build, two bases.
const base = process.env.TAURI === "1" ? "./" : "/";

export default defineConfig({
	base,
	plugins: [react(), tailwindcss()],
	build: {
		outDir: "dist",
		sourcemap: true,
	},
});

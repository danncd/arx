import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ command }) => ({
    root: fileURLToPath(new URL(".", import.meta.url)),
    base: "./",
    plugins: [
        react(),
        {
            name: "development-csp",
            transformIndexHtml(html) {
                return command === "serve"
                    ? html.replace(
                          "script-src 'self';",
                          "script-src 'self' 'unsafe-inline';",
                      )
                    : html;
            },
        },
    ],
    clearScreen: false,
    server: { host: "127.0.0.1", strictPort: true, port: 3006 },
    build: { target: "es2022", outDir: "../.build/renderer", emptyOutDir: true },
}));

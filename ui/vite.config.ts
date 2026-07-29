// ui/vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) }
  },
  build: {
    outDir: "dist",
    sourcemap: false,
    chunkSizeWarningLimit: 700,
    rollupOptions: {
      output: {
        // Function form (vite 8 / rolldown no longer accepts the object form).
        // Same vendor split as before, keyed on the package path in node_modules.
        manualChunks(id) {
          if (!id.includes("node_modules")) return;
          const groups: Record<string, string[]> = {
            "vendor-react": ["react", "react-dom", "react-router"],
            "vendor-editor": ["@monaco-editor/react", "monaco-editor"],
            "vendor-ui": ["lucide-react", "clsx", "tailwind-merge"],
            "vendor-terminal": ["@xterm/xterm", "@xterm/addon-fit"],
          };
          for (const [chunk, pkgs] of Object.entries(groups)) {
            if (pkgs.some((p) => id.includes(`/node_modules/${p}/`))) return chunk;
          }
        },
      }
    }
  }
});

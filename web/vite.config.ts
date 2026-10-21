import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import { configDefaults } from "vitest/config";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

// Docker receives the Git-derived version as a build argument, without .git.
const version = process.env.MOJI_BUILD_VERSION?.trim() || execFileSync(
  "sh", [fileURLToPath(new URL("../scripts/version.sh", import.meta.url))],
  { encoding: "utf8" },
).trim();

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const apiTarget = env.VITE_MOJI_API_TARGET || "http://127.0.0.1:10000";

  return {
    plugins: [react(), {
      name: "moji-version",
      generateBundle() {
        this.emitFile({ type: "asset", fileName: "version.json", source: JSON.stringify({ version }) + "\n" });
      }
    }],
    define: { __MOJI_VERSION__: JSON.stringify(version) },
    build: {
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (!id.includes("node_modules")) return undefined;
            if (id.includes("i18next")) return "vendor-i18n";
            if (id.includes("urql") || id.includes("wonka") || id.includes("graphql")) return "vendor-graphql";
            if (id.includes("/react/") || id.includes("react-dom") || id.includes("react-router") || id.includes("scheduler")) return "vendor-react";
            return undefined;
          }
        }
      }
    },
    test: {
      exclude: [...configDefaults.exclude, "tests/visual/**"]
    },
    server: {
      port: 5173,
      proxy: {
        "/graphql": { target: apiTarget, ws: true },
        "/healthz": apiTarget,
        "/api": apiTarget
      }
    }
  };
});

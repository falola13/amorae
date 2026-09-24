import { resolve } from "node:path";
import { defineConfig } from "vitest/config";

// Q-15 chose Vitest for units. It needs the same `@/` that tsconfig defines,
// and nothing else: what is tested here is plain functions, so there is no
// DOM environment and no setup file to keep in step with the app.
export default defineConfig({
  resolve: { alias: { "@": resolve(import.meta.dirname, "src") } },
  test: { include: ["src/**/*.test.ts"] },
});

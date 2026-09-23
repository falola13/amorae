// Regenerate apps/web/package-lock.json inside Linux.
//
// npm's lockfile is meant to be cross-platform, and for this dependency tree
// it isn't. `@tailwindcss/oxide-wasm32-wasi` declares six bundleDependencies;
// running `npm install` on Windows writes a lock that omits two of them
// (@emnapi/core and @emnapi/runtime). `npm install` never notices, because it
// resolves what is missing as it goes. `npm ci` does notice — it validates the
// tree strictly — so CI fails on Linux with "Missing: @emnapi/core from lock
// file" while everything looks fine locally.
//
// So the lock is generated where it is consumed. Run this instead of
// `npm install` in apps/web whenever a dependency changes, and commit the
// result. If it drifts, CI catches it: that is what `npm ci` is for.
import { execFileSync } from "node:child_process";
import { resolve } from "node:path";

const dir = resolve(import.meta.dirname, "..", "apps", "web");

try {
  execFileSync(
    "docker",
    // prettier-ignore
    [
      "run", "--rm",
      "-v", `${dir}:/app`,
      "-w", "/app",
      "node:24-slim",
      "npm", "install", "--package-lock-only", "--no-audit", "--no-fund",
    ],
    { stdio: "inherit" },
  );
  console.log("\napps/web/package-lock.json regenerated on Linux. Commit it.");
} catch (error) {
  console.error(
    "\nCould not regenerate the lockfile. Docker needs to be running: this has " +
      "to happen on Linux, which is where CI installs it.",
  );
  process.exit(error.status ?? 1);
}

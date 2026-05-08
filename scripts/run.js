#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
// execFileSync executes a file directly without invoking a shell.
// It is NOT equivalent to shell_exec: no shell interpolation occurs and
// arguments are passed as a plain array, making shell-injection impossible.
const { execFileSync } = require("child_process");

const ext = process.platform === "win32" ? ".exe" : "";

// Resolve to an absolute, normalised path so the boundary is unambiguous.
const binDir = path.resolve(path.join(__dirname, "..", "bin"));
const binaryPath = path.resolve(path.join(binDir, `md2wechat${ext}`));

// Belt-and-suspenders assertion: the resolved path must stay inside binDir.
// binaryPath is constructed entirely from fixed constants (not user input),
// so this guard can only fire if __dirname itself is somehow manipulated.
if (!binaryPath.startsWith(binDir + path.sep)) {
  console.error(
    "Internal error: binary path is outside the expected bin/ directory."
  );
  process.exit(1);
}

if (!fs.existsSync(binaryPath)) {
  console.error(
    "md2wechat binary is missing. Reinstall with `npm install -g @geekjourneyx/md2wechat`."
  );
  process.exit(1);
}

try {
  // Arguments are forwarded as an array — no shell expansion takes place.
  execFileSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });
} catch (error) {
  if (typeof error.status === "number") {
    process.exit(error.status);
  }

  console.error(`Failed to launch md2wechat: ${error.message}`);
  process.exit(1);
}

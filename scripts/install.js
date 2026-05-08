const crypto = require("crypto");
const fs = require("fs");
const http = require("http");
const https = require("https");
const os = require("os");
const path = require("path");
const { fileURLToPath } = require("url");

const pkg = require("../package.json");

const VERSION = pkg.version;
const REPO = "geekjourneyx/md2wechat-skill";
const PACKAGE_NAME = pkg.name;

const isWindows = process.platform === "win32";
const TARGETS = {
  darwin: {
    x64: "md2wechat-darwin-amd64",
    arm64: "md2wechat-darwin-arm64",
  },
  linux: {
    x64: "md2wechat-linux-amd64",
    arm64: "md2wechat-linux-arm64",
  },
  win32: {
    x64: "md2wechat-windows-amd64.exe",
  },
};
const releaseBaseUrl =
  process.env.MD2WECHAT_RELEASE_BASE_URL ||
  `https://github.com/${REPO}/releases/download/v${VERSION}`;
const assetName = TARGETS[process.platform]?.[process.arch];
const binaryName = isWindows ? "md2wechat.exe" : "md2wechat";
const binDir = path.join(__dirname, "..", "bin");
const destination = path.join(binDir, binaryName);

if (!assetName) {
  console.error(
    [
      `Unsupported platform for ${PACKAGE_NAME}: ${process.platform}-${process.arch}`,
      "Supported npm install targets are:",
      "  - darwin-x64",
      "  - darwin-arm64",
      "  - linux-x64",
      "  - linux-arm64",
      "  - win32-x64",
    ].join("\n")
  );
  process.exit(1);
}

function hasScheme(value) {
  return (
    /^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(value) && !path.win32.isAbsolute(value)
  );
}

function resolveAssetLocation(base, name) {
  if (!hasScheme(base)) {
    return path.join(base, name);
  }

  return base.endsWith("/") ? `${base}${name}` : `${base}/${name}`;
}

function downloadToFile(source, destinationPath) {
  if (!hasScheme(source)) {
    fs.copyFileSync(source, destinationPath);
    return Promise.resolve();
  }

  if (source.startsWith("file://")) {
    fs.copyFileSync(fileURLToPath(source), destinationPath);
    return Promise.resolve();
  }

  return new Promise((resolve, reject) => {
    const client = source.startsWith("https:") ? https : http;

    client
      .get(source, (response) => {
        if (
          (response.statusCode === 301 || response.statusCode === 302) &&
          response.headers.location
        ) {
          response.resume();
          downloadToFile(response.headers.location, destinationPath).then(
            resolve,
            reject
          );
          return;
        }

        if (response.statusCode !== 200) {
          response.resume();
          reject(
            new Error(
              `download failed with status ${response.statusCode}: ${source}`
            )
          );
          return;
        }

        // destinationPath is always inside a mkdtempSync-created temp
        // directory; assetName comes from a hardcoded platform whitelist.
        const file = fs.createWriteStream(destinationPath);
        response.pipe(file);
        file.on("finish", () => {
          file.close(resolve);
        });
        file.on("error", reject);
      })
      .on("error", reject);
  });
}

function sha256(filePath) {
  const hash = crypto.createHash("sha256");
  hash.update(fs.readFileSync(filePath));
  return hash.digest("hex");
}

function expectedChecksum(checksumsPath, filename) {
  const line = fs
    .readFileSync(checksumsPath, "utf8")
    .split(/\r?\n/)
    .find((entry) => entry.trim().endsWith(` ${filename}`));

  if (!line) {
    throw new Error(`checksums.txt does not contain an entry for ${filename}`);
  }

  return line.trim().split(/\s+/)[0].toLowerCase();
}

async function install() {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "md2wechat-npm-"));
  const downloadedBinary = path.join(tmpDir, assetName);
  const checksumsPath = path.join(tmpDir, "checksums.txt");

  try {
    fs.mkdirSync(binDir, { recursive: true });

    await downloadToFile(
      resolveAssetLocation(releaseBaseUrl, assetName),
      downloadedBinary
    );
    await downloadToFile(
      resolveAssetLocation(releaseBaseUrl, "checksums.txt"),
      checksumsPath
    );

    const expected = expectedChecksum(checksumsPath, assetName);
    const actual = sha256(downloadedBinary);
    if (expected !== actual) {
      throw new Error(`checksum mismatch for ${assetName}`);
    }

    fs.copyFileSync(downloadedBinary, destination);
    if (!isWindows) {
      fs.chmodSync(destination, 0o755);
    }

    console.log(
      `Installed md2wechat ${VERSION} from ${resolveAssetLocation(releaseBaseUrl, assetName)}`
    );
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

install().catch((error) => {
  console.error(`Failed to install ${PACKAGE_NAME}: ${error.message}`);
  process.exit(1);
});

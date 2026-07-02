#!/usr/bin/env node
/**
 * install.js — @supcode/cli postinstall hook
 *
 * Downloads the correct Go binary for the current platform
 * and installs it to the npm package's node_modules/.bin directory.
 */

const fs = require("fs");
const path = require("path");
const https = require("https");
const { createHash } = require("crypto");
const { execSync } = require("child_process");

const REPO = "supcode/supcode";
const VERSION = process.env.SUPCODE_VERSION || "latest";
const UNINSTALL = process.argv.includes("--uninstall");

const PLATFORM_MAP = {
  darwin: { os: "macOS", ext: "tar.gz" },
  linux: { os: "linux", ext: "tar.gz" },
  win32: { os: "windows", ext: "zip" },
};

const ARCH_MAP = {
  x64: "x86_64",
  arm64: "arm64",
};

function log(msg) {
  console.log(`[supcode] ${msg}`);
}

function warn(msg) {
  console.warn(`[supcode] WARNING: ${msg}`);
}

function getPlatform() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];
  if (!platform) throw new Error(`Unsupported platform: ${process.platform}`);
  if (!arch) throw new Error(`Unsupported architecture: ${process.arch}`);
  return { ...platform, arch };
}

function fetch(url) {
  return new Promise((resolve, reject) => {
    https.get(url, { timeout: 30000 }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return resolve(fetch(res.headers.location));
      }
      if (res.statusCode !== 200) {
        reject(new Error(`HTTP ${res.statusCode}: ${url}`));
        return;
      }
      const chunks = [];
      res.on("data", (c) => chunks.push(c));
      res.on("end", () => resolve(Buffer.concat(chunks)));
    }).on("error", reject);
  });
}

async function getLatestVersion() {
  const apiUrl = `https://api.github.com/repos/${REPO}/releases/latest`;
  const data = await fetch(apiUrl);
  const release = JSON.parse(data.toString());
  return release.tag_name.replace(/^v/, "");
}

async function main() {
  const plat = getPlatform();

  if (UNINSTALL) {
    const binDir = path.join(__dirname, "..", "node_modules", ".bin");
    const binPath = path.join(binDir, "supcode" + (plat.ext === "zip" ? ".exe" : ""));
    if (fs.existsSync(binPath)) {
      fs.unlinkSync(binPath);
      log("Removed supcode binary");
    }
    return;
  }

  const version = VERSION === "latest" ? await getLatestVersion() : VERSION;
  const archiveName = `supcode_${version}_${plat.os}_${plat.arch}.${plat.ext}`;
  const downloadUrl = `https://github.com/${REPO}/releases/download/v${version}/${archiveName}`;

  const binDir = path.join(__dirname, "..", "node_modules", ".bin");
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  const binName = "supcode" + (plat.ext === "zip" ? ".exe" : "");
  const binPath = path.join(binDir, binName);

  log(`Downloading supcode v${version} for ${plat.os} ${plat.arch}...`);

  const archiveData = await fetch(downloadUrl);
  log(`Downloaded ${(archiveData.length / 1024 / 1024).toFixed(1)} MB`);

  // Verify checksum
  try {
    const checksumUrl = `${downloadUrl}.sha256`;
    const checksumData = await fetch(checksumUrl);
    const expectedHash = checksumData.toString().trim().split(" ")[0];
    const actualHash = createHash("sha256").update(archiveData).digest("hex");
    if (expectedHash !== actualHash) {
      warn(`Checksum mismatch (expected ${expectedHash}, got ${actualHash}), continuing...`);
    } else {
      log("Checksum verified");
    }
  } catch {
    warn("Checksum verification unavailable, skipping");
  }

  // Extract
  const tmpDir = fs.mkdtempSync(path.join(__dirname, "..", ".tmp-"));
  try {
    if (plat.ext === "zip") {
      execSync(`tar -xf "${archiveData}" -C "${tmpDir}"`, { stdio: "pipe" });
    } else {
      // For tar.gz, write to temp file first
      const tmpArchive = path.join(tmpDir, "archive.tar.gz");
      fs.writeFileSync(tmpArchive, archiveData);
      execSync(`tar -xzf "${tmpArchive}" -C "${tmpDir}"`, { stdio: "pipe" });
    }

    const extractedBin = fs.readdirSync(tmpDir).find((f) => f.startsWith("supcode"));
    if (!extractedBin) throw new Error("Binary not found in archive");

    fs.copyFileSync(path.join(tmpDir, extractedBin), binPath);
    fs.chmodSync(binPath, 0o755);

    log(`Installed to ${binPath}`);
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

main().catch((err) => {
  warn(`Installation failed: ${err.message}`);
  warn("You can manually download supcode from https://github.com/supcode/supcode/releases");
  process.exit(0); // Don't break npm install on failure
});

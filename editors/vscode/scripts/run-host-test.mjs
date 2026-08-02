import {runTests} from "@vscode/test-electron";
import {resolve} from "node:path";
import process from "node:process";

const versionIndex = process.argv.indexOf("--version");
const version = versionIndex >= 0 ? process.argv[versionIndex + 1] : "1.131.0";
if (!/^1\.\d+\.\d+$/.test(version)) {
  throw new Error(`invalid pinned VS Code version: ${version}`);
}

const extensionDevelopmentPath = resolve(import.meta.dirname, "..");
await runTests({
  version,
  extensionDevelopmentPath,
  extensionTestsPath: resolve(extensionDevelopmentPath, "test", "host", "index.cjs"),
  extensionTestsEnv: {BATCHWEAVER_EXPECTED_VSCODE: version},
  launchArgs: ["--disable-updates", "--skip-welcome", "--skip-release-notes"],
});

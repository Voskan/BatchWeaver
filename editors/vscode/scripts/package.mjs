import {copyFile, readFile, rm, writeFile} from "node:fs/promises";
import {spawnSync} from "node:child_process";
import {resolve} from "node:path";

const extensionRoot = process.cwd();
const manifest = JSON.parse(await readFile(resolve(extensionRoot, "package.json"), "utf8"));
if (manifest.name !== "batchweaver-vscode") {
  throw new Error("refusing to package outside editors/vscode");
}
const generated = ["LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.md"];
try {
  for (const name of ["LICENSE", "NOTICE"]) {
    await copyFile(resolve(extensionRoot, "../..", name), resolve(extensionRoot, name));
  }
  const lock = JSON.parse(await readFile(resolve(extensionRoot, "package-lock.json"), "utf8"));
  const queue = Object.keys(lock.packages[""].dependencies ?? {}).map((name) => `node_modules/${name}`);
  const seen = new Set();
  const runtime = [];
  while (queue.length) {
    const packagePath = queue.shift();
    if (seen.has(packagePath)) continue;
    seen.add(packagePath);
    const metadata = JSON.parse(await readFile(resolve(extensionRoot, packagePath, "package.json"), "utf8"));
    runtime.push({name: metadata.name, version: metadata.version, license: metadata.license ?? "NOASSERTION"});
    for (const name of Object.keys(metadata.dependencies ?? {})) {
      const nested = `${packagePath}/node_modules/${name}`;
      queue.push(lock.packages[nested] ? nested : `node_modules/${name}`);
    }
  }
  runtime.sort((a, b) => a.name.localeCompare(b.name));
  const notices = ["# VS Code Extension Third-Party Notices", "", "| Component | Version | License |", "| --- | --- | --- |", ...runtime.map((item) => `| ${item.name} | ${item.version} | ${item.license} |`), ""].join("\n");
  await writeFile(resolve(extensionRoot, "THIRD_PARTY_NOTICES.md"), notices, "utf8");
  const result = spawnSync(
    resolve(extensionRoot, "node_modules", ".bin", process.platform === "win32" ? "vsce.cmd" : "vsce"),
    ["package", "--out", "batchweaver-vscode.vsix"],
    {cwd: extensionRoot, encoding: "utf8", stdio: "inherit", shell: false}
  );
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`vsce exited with status ${result.status}`);
} finally {
  for (const name of generated) await rm(resolve(extensionRoot, name), {force: true});
}

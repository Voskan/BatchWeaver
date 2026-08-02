import assert from "node:assert/strict";
import {readFile} from "node:fs/promises";

const root = new URL("../", import.meta.url);
const manifest = JSON.parse(await readFile(new URL("package.json", root), "utf8"));
const source = await readFile(new URL("src/extension.ts", root), "utf8");

assert.equal(manifest.private, true, "the beta extension must remain non-publishing");
assert.equal(manifest.version, "0.1.0-beta.3");
assert.equal(manifest.license, "Apache-2.0");
assert.match(manifest.engines.vscode, /^\^1\.85\.0$/);
assert.equal(manifest.engines.node, ">=22");
const contributedCommands = manifest.contributes.commands.map(({command}) => command).sort();
assert.deepEqual(contributedCommands, [
  "batchweaver.doctor",
  "batchweaver.openLogs",
  "batchweaver.previewTransformation",
  "batchweaver.proveCandidate",
  "batchweaver.restartServer",
  "batchweaver.scanWorkspace",
  "batchweaver.showOperationGraph",
]);
assert.ok(source.includes("middleware:"), "the language-client middleware is missing");
assert.ok(source.includes("executeCommand:"), "server command results are not handled");
for (const command of [
  "batchweaver.scanWorkspace",
  "batchweaver.previewTransformation",
  "batchweaver.proveCandidate",
  "batchweaver.showOperationGraph",
  "batchweaver.doctor",
]) {
  assert.ok(
    !source.includes(`commands.registerCommand("${command}"`),
    `server-advertised command ${command} must not be registered twice`,
  );
}
for (const command of ["batchweaver.openLogs", "batchweaver.restartServer"]) {
  assert.ok(
    source.includes(`commands.registerCommand("${command}"`),
    `editor-local command ${command} must be registered by the extension`,
  );
}
for (const setting of Object.keys(manifest.contributes.configuration.properties)) {
  const shortName = setting.replace("batchweaver.", "");
  assert.ok(source.includes(shortName), `setting ${setting} is not consumed in extension.ts`);
}
console.log(`verified ${manifest.contributes.commands.length} commands and ${Object.keys(manifest.contributes.configuration.properties).length} settings`);

import assert from "node:assert/strict";
import {readFile} from "node:fs/promises";

const root = new URL("../", import.meta.url);
const manifest = JSON.parse(await readFile(new URL("package.json", root), "utf8"));
const source = await readFile(new URL("src/extension.ts", root), "utf8");

assert.equal(manifest.private, true, "the RC extension must remain non-publishing");
assert.equal(manifest.version, "0.1.0-rc.1");
assert.equal(manifest.license, "Apache-2.0");
assert.match(manifest.engines.vscode, /^\^1\.85\.0$/);
assert.equal(manifest.engines.node, ">=22");
for (const contribution of manifest.contributes.commands) {
  assert.ok(source.includes(contribution.command), `command ${contribution.command} is not registered in extension.ts`);
}
for (const setting of Object.keys(manifest.contributes.configuration.properties)) {
  const shortName = setting.replace("batchweaver.", "");
  assert.ok(source.includes(shortName), `setting ${setting} is not consumed in extension.ts`);
}
console.log(`verified ${manifest.contributes.commands.length} commands and ${Object.keys(manifest.contributes.configuration.properties).length} settings`);

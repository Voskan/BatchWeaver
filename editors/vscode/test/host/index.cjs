const assert = require("node:assert/strict");
const vscode = require("vscode");

async function run() {
  assert.equal(vscode.version, process.env.BATCHWEAVER_EXPECTED_VSCODE);
  await vscode.workspace
    .getConfiguration("batchweaver")
    .update("enabled", false, vscode.ConfigurationTarget.Global);

  const extension = vscode.extensions.getExtension("batchweaver.batchweaver-vscode");
  assert.ok(extension, "BatchWeaver extension is not installed in the test host");
  await extension.activate();
  assert.equal(extension.isActive, true, "BatchWeaver extension did not activate");

  const commands = new Set(await vscode.commands.getCommands(true));
  for (const command of ["batchweaver.openLogs", "batchweaver.restartServer"]) {
    assert.ok(commands.has(command), `activated extension did not register ${command}`);
  }

  await vscode.workspace
    .getConfiguration("batchweaver")
    .update("enabled", undefined, vscode.ConfigurationTarget.Global);
}

module.exports = {run};

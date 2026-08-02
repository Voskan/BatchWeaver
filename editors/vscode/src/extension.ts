// BatchWeaver VS Code extension.
//
// It starts the BatchWeaver language server in sidecar mode (alongside gopls) or
// proxy mode (as the Go language server, delegating standard features to gopls),
// wires the status bar, output channel, and commands, and honors workspace
// trust: in an untrusted workspace it does not start the server, gopls, or any
// child process.

import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;
let status: vscode.StatusBarItem;
let output: vscode.OutputChannel;

export function activate(context: vscode.ExtensionContext): void {
  output = vscode.window.createOutputChannel("BatchWeaver");
  status = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
  status.text = "BatchWeaver";
  status.command = "batchweaver.doctor";
  context.subscriptions.push(output, status);
  registerLocalCommands(context);

  if (!vscode.workspace.isTrusted) {
    status.text = "BatchWeaver: restricted (untrusted)";
    status.show();
    output.appendLine(
      "Workspace is not trusted; BatchWeaver will not start the language server, gopls, or any child process."
    );
    context.subscriptions.push(
      vscode.workspace.onDidGrantWorkspaceTrust(() => void startClient(context))
    );
    return;
  }
  void startClient(context);
}

async function startClient(context: vscode.ExtensionContext): Promise<void> {
  const cfg = vscode.workspace.getConfiguration("batchweaver");
  if (!cfg.get<boolean>("enabled", true)) {
    return;
  }
  const serverPath = cfg.get<string>("server.path", "batchweaver");
  const mode = cfg.get<string>("mode", "sidecar");
  const goplsPath = cfg.get<string>("gopls.path", "gopls");

  const args = ["lsp", "--stdio"];
  if (mode === "proxy") {
    args.push("--proxy-gopls", `--gopls-command=${goplsPath}`);
  }

  const serverOptions: ServerOptions = {
    command: serverPath,
    args,
    transport: TransportKind.stdio,
  };
  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "go" }],
    outputChannel: output,
    // BatchWeaver diagnostics use the "batchweaver" source and never clear
    // gopls diagnostics.
    diagnosticCollectionName: "batchweaver",
    middleware: {
      // vscode-languageclient registers the command IDs advertised by the
      // server. Registering the same IDs directly in this extension prevents
      // the client from completing initialization, so render the server result
      // in middleware instead of installing a second command handler.
      executeCommand: async (command, args, next) => {
        const result = await next(command, args);
        await showCommandResult(command, result);
        return result;
      },
    },
  };

  client = new LanguageClient("batchweaver", "BatchWeaver", serverOptions, clientOptions);
  try {
    await client.start();
    status.text = `BatchWeaver: ${mode}`;
    status.show();
    context.subscriptions.push(client);
  } catch (err) {
    output.appendLine(`Failed to start BatchWeaver server: ${String(err)}`);
    status.text = "BatchWeaver: error";
    status.show();
  }
}

function registerLocalCommands(context: vscode.ExtensionContext): void {
  context.subscriptions.push(
    vscode.commands.registerCommand("batchweaver.openLogs", () => output.show()),
    vscode.commands.registerCommand("batchweaver.restartServer", async () => {
      await client?.stop();
      await startClient(context);
    })
  );
}

async function showCommandResult(command: string, result: unknown): Promise<void> {
  if (result === undefined || result === null) {
    return;
  }
  const doc = await vscode.workspace.openTextDocument({
    content: typeof result === "string" ? result : JSON.stringify(result, null, 2),
    language: command === "batchweaver.showOperationGraph" ? "dot" : "plaintext",
  });
  await vscode.window.showTextDocument(doc, { preview: true });
}

export async function deactivate(): Promise<void> {
  await client?.stop();
}

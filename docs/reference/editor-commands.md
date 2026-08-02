# Editor commands reference

Stable command IDs exposed through `workspace/executeCommand` and the VS Code
extension. All commands are read-only; none mutate source.

| Command ID | Title | Result |
| --- | --- | --- |
| `batchweaver.scanWorkspace` | Scan Workspace | Scan summary text. |
| `batchweaver.previewTransformation` | Preview Transformation | Candidate preview text. |
| `batchweaver.proveCandidate` | Prove Candidate | Proof/preview text. |
| `batchweaver.showOperationGraph` | Show Operation Graph | DOT graph text. |
| `batchweaver.doctor` | Run Doctor | Environment summary text. |

The VS Code extension adds editor-only commands `batchweaver.restartServer` and
`batchweaver.openLogs`.

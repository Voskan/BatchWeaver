# LSP diagnostics reference

Editor/LSP-layer diagnostics occupy the `BW9xxx` range, kept distinct from
analysis (`BW3xxx`), proof (`BW5xxx`), backend adapter (`BW6xxx`), network
adapter (`BW7xxx`), and adaptive (`BW8xxx`) codes. The editor specification
illustrated these in the `BW8xxx` range, which the adaptive layer already owns;
they are renumbered here into `BW9xxx` to preserve the repository rule that
diagnostic ranges are distinct per stage.

| Code | Meaning |
| --- | --- |
| `BW9001` | editor snapshot is stale |
| `BW9002` | workspace edit cannot be applied safely |
| `BW9003` | gopls version is incompatible |
| `BW9004` | LSP client lacks required WorkspaceEdit support |
| `BW9005` | proxy capability merge failed |
| `BW9006` | daemon protocol is incompatible |
| `BW9007` | deep analysis canceled by newer snapshot |
| `BW9008` | untrusted workspace blocks command execution |
| `BW9009` | virtual document expired |
| `BW9010` | multiple BatchWeaver servers own this workspace |

The batching-opportunity diagnostic (`BW1001`) and all analysis/proof/adapter
diagnostics keep their existing codes when surfaced in the editor, with the
`batchweaver` source.

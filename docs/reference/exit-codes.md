# Exit codes

BatchWeaver commands return stable process exit codes.

| Code | Meaning |
| --- | --- |
| 0 | success |
| 1 | transformation, validation, or internal runtime error |
| 2 | invalid CLI usage or configuration |
| 3 | configuration is invalid |
| 4 | no configuration file was found |
| 5 | stale analysis, proof, source, or plan |
| 6 | the underlying Go command reported failure |
| 7 | materialization or revert conflict |

`batchweaver build`, `test`, and `run` return exit code 6 when the underlying Go
command fails, so an application test failure is never reported as a BatchWeaver
internal error. Materialization and revert conflicts return exit code 7.

# Public API Freeze Review

## Status

**Candidate inventory complete; stable freeze not approved.** The repository
has an exact exported-Go baseline, but no published prerelease compatibility
window or maintainer v1 approval exists.

## Go package inventory

The canonical element-level inventory is
`internal/release/testdata/public-api.txt` and is verified by
`TestPublicAPIBaseline`. At the audited commit it contains 1,158 object and
method records:

| Package | Records | Classification before v1 |
| --- | ---: | --- |
| module root | 60 | stable candidate |
| `adapters/gqlgen` | 9 | experimental candidate; gqlgen v0.17.94 |
| `adapters/grpcgo` | 11 | experimental candidate; grpc-go v1.83.0 |
| `adapters/pgxv5` | 6 | experimental candidate; pgx v5.10.0 |
| `adapters/redisv9` | 11 | experimental candidate; go-redis v9.21.0 |
| `bridge` | 6 | experimental, versioned ABI |
| `config` | 50 | stable candidate, schema 1 |
| `diagnostics` | 78 | stable candidate, diagnostic policy |
| `operation` | 761 | stable candidate; large surface requires review |
| `runtime` | 166 | stable candidate; concurrency contract requires review |

“Stable candidate” means documented and baseline-checked, not approved stable.
Compiler, proof, transformation, adaptive, daemon, and editor implementation
APIs remain internal. The four concrete client packages are public so
applications can construct typed providers without reflection.

## Other compatibility surfaces

- CLI hierarchy and flags: `batchweaver help` plus command-specific help.
- Exit codes: `docs/reference/exit-codes.md`.
- Diagnostics: `docs/reference/diagnostic-codes.md`.
- Configuration: schema 1, strict unknown-key policy.
- Bridge ABI: `batchweaver.bridge/v1alpha1`.
- Analysis/proof/transform: independent `v1alpha1` schemas.
- Adapter verification and manifests: independent `v1alpha1` schemas.
- Profile/controller/wave/replay: independent `v1alpha1` schemas.
- Daemon protocol: `batchweaver.daemon/v1alpha1`.
- Generated code and caches: invalidated on incompatible schema, ABI, contract,
  source anchor, or toolchain changes.

The machine-readable summary is `release/api-inventory-v1.json`.

## Approval blockers

- the new concrete adapter APIs have not been exercised in a published
  prerelease compatibility window or independent downstream modules;
- no prerelease migration has been tested from a public version;
- the `operation` and `runtime` surfaces need explicit maintainer review;
- the bridge and artifact schemas remain alpha;
- compatibility CI cannot compare against a latest stable tag because none
  exists.

No identifier is declared stable v1 by this document.

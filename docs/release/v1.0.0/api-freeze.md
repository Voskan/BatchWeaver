# Public API Freeze Review

## Status

**Candidate inventory complete; stable freeze not approved.** The repository
has an exact exported-Go baseline, but no published prerelease compatibility
window or maintainer v1 approval exists.

## Go package inventory

The canonical element-level inventory is
`internal/release/testdata/public-api.txt` and is verified by
`TestPublicAPIBaseline`. At the audited commit it contains 1,121 object and
method records:

| Package | Records | Classification before v1 |
| --- | ---: | --- |
| module root | 60 | stable candidate |
| `bridge` | 6 | experimental, versioned ABI |
| `config` | 50 | stable candidate, schema 1 |
| `diagnostics` | 78 | stable candidate, diagnostic policy |
| `operation` | 761 | stable candidate; large surface requires review |
| `runtime` | 166 | stable candidate; concurrency contract requires review |

“Stable candidate” means documented and baseline-checked, not approved stable.
Compiler, proof, transformation, adapter, adaptive, daemon, and editor Go APIs
remain internal.

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

- no published beta API has been exercised by downstream modules;
- no prerelease migration has been tested from a public version;
- the `operation` and `runtime` surfaces need explicit maintainer review;
- the bridge and artifact schemas remain alpha;
- compatibility CI cannot compare against a latest stable tag because none
  exists.

No identifier is declared stable v1 by this document.

# Production-like Soak, Fuzz, Leak, and Fault Campaigns

BatchWeaver has a scheduled and manually dispatchable extended campaign at
`.github/workflows/production-campaign.yml`. It complements pull-request CI; it
does not replace deterministic unit, race, compatibility, security, or release
checks.

## Campaign shape

Every campaign runs against one exact commit with pinned Go 1.26.5, Node
22.22.0, GitHub Actions revisions, VS Code 1.131.0, and the module versions in
`go.sum`.

- Eleven parallel fuzz categories cover configuration, schemas, proof,
  transformations, SQL synthesis, protocol data, LSP framing, daemon discovery,
  profiles, release manifests, and migration inventories.
- Mixed phases exercise compiler analysis, runtime scheduling, concrete client
  adapters, daemon/cache lifecycle, adaptive control, and a real VS Code
  Extension Host.
- The production-like soak combines deterministic heavy-tail traffic,
  duplicates, tenant skew, deadlines, phase changes, differential scalar/batch
  comparison, adaptive histograms, and composite/join SQL synthesis under the
  race detector.
- Leak budgets cover goroutines, heap growth, timers, files, sockets, row/body/
  stream closers, and child processes.
- Fault injection covers timeout, connection reset, partial and malformed
  responses, corruption, permissions, process crash, truncated downloads, and
  invalid signatures.
- An unpublished release is built, verified, and reproduced as a separate
  phase.

The Saturday schedule uses 30 minutes per fuzz category and a 60-minute soak.
Manual dispatch accepts explicit fuzz and soak durations. Matrix fuzzing runs in
parallel; a campaign is successful only when the final policy job validates all
required categories.

## Evidence contract

Each phase is wrapped by `.github/scripts/campaign-phase.go`, which preserves its
log and emits `batchweaver.campaign-phase/v1alpha1` JSON containing:

- exact commit and Actions run URL;
- command, pinned environment, campaign seed, and Go fuzz target/corpus digest;
- start, finish, elapsed milliseconds, exit status, and failure reason;
- recorder goroutine, heap, descriptor, and child maximum-RSS deltas.

The policy job downloads every phase artifact and
`.github/scripts/validate-campaign-evidence.go` rejects missing categories,
mixed commits, incomplete metadata, and any failed phase. The combined
`production-campaign-<commit>` artifact is retained for 90 days; phase logs and
generated fuzz corpora are retained for 30 days.

## Reproduction

The bounded local smoke equivalents are:

```bash
make campaign-smoke
BATCHWEAVER_SOAK_DURATION=30m BATCHWEAVER_CAMPAIGN_SEED=1119513617 \
  go test -race ./internal/assurance -run '^TestProductionLikeSoak$' \
  -count=1 -timeout 35m
go test ./internal/adapter -run '^$' -fuzz '^FuzzSQLSynthesis$' -fuzztime 15m
```

Use the Actions workflow for retained hosted evidence. A short local run proves
only that the harness works. Long-term stability evidence requires
multiple successful hosted campaigns at the exact final candidate commit;
until those runs exist, no long-term stability claim is supported.

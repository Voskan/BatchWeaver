# Assumptions and trust

The proof engine operates under a documented trust model. It separates what
follows from Go semantics, what follows from BatchWeaver contracts, what is merely
assumed, what is impossible, and what remains unknown.

## Trusted sources

- the Go 1.26 language specification and memory model;
- documented standard-library synchronization semantics modeled by the analysis
  stage;
- validated operation contracts;
- analysis facts produced from the selected build context;
- validated BatchWeaver configuration;
- models versioned with the proof engine.

## Assumption-based sources

- user annotations and adapter metadata;
- claims that a custom wrapper is pure or synchronization-equivalent;
- claims about remote backend behavior;
- claims about transaction, authorization, or consistency behavior not inferable
  from code.

## Assumption rules

- Every assumption has a stable ID, exact symbol scope, declared facts, digest,
  and origin.
- An assumption satisfies only the obligation types that reference the fact it
  declares. A side-effect-free-read assertion, for example, does not satisfy a
  transaction-partition obligation.
- Assumptions are never applied automatically, and a wildcard (workspace-wide)
  assertion is rejected unless an explicit unsafe mode is enabled.
- No assumption overrides a hard Go-language fact.
- Reports distinguish inferred evidence from assumed evidence, and every applied
  assumption appears in the certificate.

## The data-race-free prerequisite

Static analysis cannot prove the absence of data races. The proof model may assume
data-race freedom, but that assumption (`BW-A-RACEFREE`) is recorded in every
certificate whose eligibility depends on shared-memory reasoning, and definitive
race detection is left to the Go race detector. Incorrect assumptions can
invalidate the guarantees of a certificate.

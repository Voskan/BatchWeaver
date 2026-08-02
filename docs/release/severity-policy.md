# Release-Blocking Severity

| Class | Meaning |
| --- | --- |
| blocker | Publication or use would violate semantics, isolation, integrity, or explicit policy. |
| critical | Likely severe security, data, or source-corruption impact. |
| major | Material supported-path failure without a safe transparent fallback. |
| minor | Limited impact with a documented safe workaround. |
| informational | Verified observation with no present user impact. |

Semantic mismatch, partition or transaction mixing, source corruption, secret
exposure, required-signature failure, checksum mismatch, false reproducibility,
primary compatibility failure, and packaged-install failure are blockers.

Release diagnostics occupy BW9001–BW9018: dirty source, stale generation, API or
schema break, semantic mismatch, surviving mutation, fuzz regression,
compatibility failure, performance budget, security or license finding, SBOM or
provenance failure, checksum or reproducibility failure, unverified docs,
packaged quickstart failure, and unauthorized publication respectively.

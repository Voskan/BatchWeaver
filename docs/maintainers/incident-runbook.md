# Safety and Security Incident Runbook

## Immediate response

1. record the affected version, strategy, operation, adapter, platform, and
   artifact identity without collecting unnecessary user data;
2. advise disabling the affected transformation, adapter, or active tuning mode;
3. preserve scalar/direct execution and stop new materialization;
4. use a private advisory for exploitable, tenant, authorization, transaction,
   data-corruption, or supply-chain reports;
5. invalidate affected proof, transform, generated bridge, profile, or cache
   versions when their trust basis is compromised.

## Investigation

Reproduce, minimize, compare scalar and transformed behavior, identify affected
versions, assess P0/P1 severity, add a regression test, and run differential and
mutation coverage appropriate to the failure. Do not expose private source,
credentials, payloads, tenant data, or exploit details.

## Remediation and release

Fix the root cause, verify rollback and feature disablement, prepare a compatible
patch or security release, build from a clean pinned environment, and verify all
public artifacts. Backport only according to the release policy.

## Communication

State affected versions, safe mitigations, fixed versions, artifact verification,
and whether caches/generated files must be discarded. Do not claim impact is
absent merely because reports are limited. Coordinate advisory publication only
after a fix is available.

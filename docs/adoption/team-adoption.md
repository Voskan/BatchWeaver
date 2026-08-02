# Team Adoption Checklist

1. Evaluate a pinned beta in a non-production clone.
2. Record the Go/platform/client matrix and keep scan-only mode first.
3. Review diagnostics and rejected candidates; never weaken proof policy to
   increase candidate count.
4. Add explicit providers for narrow, verified contracts.
5. Review proof evidence and transformation diffs in code review.
6. Run transformed unit, race, integration, and application tests without
   materializing source.
7. Enable runtime shadow verification and performance budgets.
8. Canary one operation and partition policy at a time.
9. Monitor correctness, fallbacks, cancellation, saturation, and SLOs.
10. Rehearse scalar fallback, cache invalidation, source revert, and binary
    downgrade before production use.

The beta has no production-validation claim. Every adopting team owns its
application-specific correctness, load, authorization, and rollback evidence.

# Governance

BatchWeaver uses a maintainer-led model. Voskan Voskanyan
([@Voskan](https://github.com/Voskan)) is the founder, current lead maintainer,
repository owner, final technical decision maker, and release authority. There
are no other maintainers at this time.

Contributors propose changes through issues and pull requests. Routine changes
use normal review; cross-cutting decisions use ADRs under `docs/adr`. Decisions
favor conservative semantics, public evidence, reversible rollout, and recorded
tradeoffs. Unresolved material disagreement is decided by the lead maintainer
and documented.

Releases require an explicit decision record, immutable version, clean required
gates, known-issue review, rollback plan, and verified publishing identity.
Security authority may embargo details and use a private fix branch. No
automation may bypass a required correctness or security review.

Maintainers have merge, release, and security responsibilities. Future
maintainers may be invited after sustained, high-quality contributions and an
explicit ownership decision. Inactive access should be removed only after
continuity is established; administration is never transferred implicitly.

Contributions are recognized through Git history and release notes when
material. The project does not create honorary or fictitious maintainer roles.

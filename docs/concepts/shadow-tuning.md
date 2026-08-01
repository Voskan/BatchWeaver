# Shadow tuning

Shadow tuning runs the adaptive controller against live or recorded profiles and
records what it *would* do, without changing any runtime setting. It is the
default mode and the required precursor to active tuning.

Shadow decisions carry the same evidence, reasons, confidence, and expected-effect
estimates as active decisions, so operators can review the controller's behavior
before granting it the ability to act. `batchweaver tune shadow` and
`batchweaver tune analyze` render shadow recommendations.

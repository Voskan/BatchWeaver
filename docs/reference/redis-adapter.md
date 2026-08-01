# Redis adapter reference

Status: cluster slot and result-mapping logic implemented and tested; concrete
go-redis client binding deferred (offline dependency).

## Command mapping

| Scalar | Batch |
| --- | --- |
| `GET` | `MGET` |
| `HGET` | `HMGET` (per hash key) |
| independent commands | pipeline |

## Cluster slots

Slots are computed with CRC-16/XMODEM modulo 16384, honoring `{tag}` hash tags.
`SlotGroups` groups keys by slot (deterministically ordered, original order within
a slot) so multi-key commands never cross slots; `SameSlot` reports whether a set
of keys is issuable as one cross-key command.

## Missing and pipeline errors

Missing keys/fields map to the operation's declared missing outcome. Pipeline
errors may be global (execution) or per-command; both are mapped without losing
the original error identity.

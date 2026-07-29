# Diagnostic Code Reference

Diagnostic codes use the format `BW<CATEGORY><NNN>`. Once a code is committed and
documented it keeps its meaning. Reserved category ranges:

```text
BWCFG001–BWCFG099  configuration syntax, loading, includes, schema
BWOP001–BWOP099    operation IDs, symbols, semantics, and contracts
BWDECL001–BWDECL099 typed declaration validation (reserved; not yet assigned)
BWCLI001–BWCLI099  command-line usage (reserved; not yet assigned)
BWINT001–BWINT099  internal consistency failures (reserved; not yet assigned)
```

## Configuration (BWCFG)

| Code | Meaning |
| ---- | ------- |
| BWCFG001 | YAML or JSON syntax error |
| BWCFG002 | Duplicate mapping key |
| BWCFG003 | More than one document in a source |
| BWCFG004 | Trailing content after a JSON document |
| BWCFG005 | Value of an unexpected kind (type mismatch) |
| BWCFG006 | Unknown enumeration value |
| BWCFG007 | Invalid duration value |
| BWCFG008 | Invalid byte-size value |
| BWCFG009 | Disallowed YAML construct (anchor, alias, tag, merge key) |
| BWCFG010 | Size or count limit exceeded |
| BWCFG011 | Failure to read or resolve an include |
| BWCFG012 | Include cycle |
| BWCFG013 | Disallowed absolute include path |
| BWCFG014 | Forbidden remote include |
| BWCFG015 | Missing required field |
| BWCFG016 | Unsupported schema version |
| BWCFG017 | No configuration file found |
| BWCFG018 | Multiple candidate configuration files |
| BWCFG019 | Conflicting operation definition |
| BWCFG020 | Configuration security violation |
| BWCFG021 | Unknown field |
| BWCFG022 | Value invalid for its field |
| BWCFG023 | Semantic validation failure |

## Operation (BWOP)

| Code | Meaning |
| ---- | ------- |
| BWOP001 | Invalid operation ID |
| BWOP002 | Invalid semantics |
| BWOP003 | Invalid result contract |
| BWOP004 | Invalid partition contract |
| BWOP005 | Invalid scheduler policy |
| BWOP006 | Invalid deduplication policy |
| BWOP007 | Invalid retry policy |
| BWOP008 | Invalid fallback policy |
| BWOP009 | Invalid key contract |
| BWOP010 | Invalid extension |
| BWOP011 | Partition/semantics incompatibility |
| BWOP012 | Deduplication incompatible with operation kind |
| BWOP013 | Retry incompatible with idempotency or result mode |
| BWOP014 | Fallback incompatible with effect or symbols |
| BWOP015 | Configuration-based operation missing scalar/batch symbols |
| BWOP016 | Invalid symbol |
| BWOP017 | Duplicate operation ID in a catalog |

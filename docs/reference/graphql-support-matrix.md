# GraphQL support matrix

| Feature | Status |
| --- | --- |
| named/anonymous operations | supported |
| query / mutation / subscription recognition | supported |
| fields, aliases, arguments, directives | supported (parsed) |
| fragment definitions, spreads, inline fragments | supported |
| resolver-wave computation | supported |
| normalized selection digest / partitioning | supported |
| error path / nullability preservation model | supported |
| one scope per operation | supported |
| gqlgen operation scope extension | supported and tested |
| gqlgen normalized field partition context | supported and tested |
| cross-operation / cross-request batching | rejected |
| subscription lifetime-wide batching | rejected |
| top-level mutation reordering | rejected |

Diagnostics use the BW71xx range.

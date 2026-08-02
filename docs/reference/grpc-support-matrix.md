# gRPC support matrix

| Feature | Status |
| --- | --- |
| explicit unary scalar-to-batch binding | supported (model + validation) |
| keyed / positional response correlation | supported |
| metadata partition policy | supported |
| status code/message/detail preservation model | supported |
| streaming contracts (client/server/bidi) | contract + lifecycle defined |
| concrete grpc-go unary provider / bufconn integration | supported and tested |
| automatic streaming RPC batching | not supported; explicit contract required |
| server-side batch method generation | not implemented (out of scope) |
| merging auth/credentials across callers | rejected |

Diagnostics use the BW72xx range.

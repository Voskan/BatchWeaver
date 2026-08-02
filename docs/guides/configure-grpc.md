# Configure gRPC

Declare an explicit batch RPC binding:

```yaml
operations:
  users.get:
    adapter:
      id: grpc-go
      scalar_method: /users.v1.UserService/GetUser
      batch_method: /users.v1.UserService/BatchGetUsers
      request_key: user_id
      batch_requests_field: requests
      response_mode: keyed
      response_key: user_id
      per_item_status: status
```

Inspect a binding and the metadata policy:

```bash
batchweaver grpc inspect --scalar=/users.v1.UserService/GetUser \
  --batch=/users.v1.UserService/BatchGetUsers --key=user_id --response-key=user_id
```

Different authorization, credentials, tenant, or routing metadata partition
callers so they never share a batch. `adapters/grpcgo.UnaryProvider` accepts an
existing generated batch request/response type and invokes the explicitly named
batch RPC through `grpc.ClientConnInterface`. Its decoder must correlate exactly
one outcome to every BatchWeaver request ID; missing or unexpected IDs fail the
provider. See [limitations](../limitations/network-adapters.md).

// Package gqlgen integrates BatchWeaver request scopes with gqlgen through its
// public extension and field-interceptor APIs.
//
// Register [ScopeExtension] with handler.Server.Use. Every GraphQL operation
// receives an isolated BatchWeaver runtime scope. Field resolvers can obtain
// normalized selection and path metadata with [FieldInfoFromContext] and use
// [PartitionFromContext] when constructing a runtime binding partition key.
package gqlgen

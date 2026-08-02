// Package pgxv5 provides typed BatchWeaver runtime providers for pgx v5.
//
// Providers accept a caller-owned [Queryer]. Passing a *pgx.Conn, pgx.Tx, or
// *pgxpool.Pool keeps connection and transaction selection explicit; the
// adapter never starts a transaction or acquires a different connection.
// Queries and decoders are supplied by the application or generated bridge,
// and key values are always passed as parameters.
package pgxv5

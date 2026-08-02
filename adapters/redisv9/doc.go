// Package redisv9 provides typed BatchWeaver runtime providers for go-redis v9.
//
// MGET requests are split by Redis Cluster hash slot before execution. HMGET
// requests are grouped by hash key. PipelineProvider supports explicit typed
// commands when no safe multi-key command exists. Every provider reconstructs
// outcomes in original request order and preserves duplicate requests.
package redisv9

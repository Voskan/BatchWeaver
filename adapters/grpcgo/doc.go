// Package grpcgo provides typed BatchWeaver runtime providers for explicit
// grpc-go batch RPCs.
//
// The adapter never invents or generates a remote batch method. Applications
// provide an existing method name plus typed request construction and response
// correlation functions. Outgoing metadata can be converted into a
// conservative runtime partition with [MetadataPartition].
package grpcgo

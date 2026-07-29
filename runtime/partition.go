package runtime

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"strings"
)

// Partition tags to keep single and composite partitions distinct and to make
// encodings unambiguous.
const (
	tagSingle    = 0x01
	tagComposite = 0x02
)

// Partition is an opaque, immutable batching boundary. Requests in different
// partitions never share a provider batch. The encoding is length-delimited so
// that component boundaries are unambiguous, and equality is exact over the
// encoded bytes, so hash collisions never merge distinct partitions.
//
// The zero Partition is invalid; construct one with PartitionFromStrings,
// PartitionFromBytes, or SinglePartition.
type Partition struct {
	encoded string
	hash    uint64
}

// newPartition builds a Partition from its encoded form.
func newPartition(encoded string) Partition {
	return Partition{encoded: encoded, hash: maphash.String(keySeed, encoded)}
}

// PartitionFromStrings builds a partition from string components. Distinct
// component groupings never collide because each component is length-delimited.
func PartitionFromStrings(values ...string) Partition {
	var b strings.Builder
	b.WriteByte(tagComposite)
	var lenbuf [binary.MaxVarintLen64]byte
	for _, v := range values {
		n := binary.PutUvarint(lenbuf[:], uint64(len(v)))
		b.Write(lenbuf[:n])
		b.WriteString(v)
	}
	return newPartition(b.String())
}

// PartitionFromBytes builds a partition from byte components, copying them into
// owned storage.
func PartitionFromBytes(values ...[]byte) Partition {
	var b strings.Builder
	b.WriteByte(tagComposite)
	var lenbuf [binary.MaxVarintLen64]byte
	for _, v := range values {
		n := binary.PutUvarint(lenbuf[:], uint64(len(v)))
		b.Write(lenbuf[:n])
		b.Write(v)
	}
	return newPartition(b.String())
}

// SinglePartition returns the partition used when an operation is not
// partitioned. It is distinct from any composite partition, including an empty
// one.
func SinglePartition() Partition {
	return newPartition(string([]byte{tagSingle}))
}

// IsZero reports whether the partition is the invalid zero value.
func (p Partition) IsZero() bool { return p.encoded == "" }

// String returns a redacted, stable-within-process token derived from a hash of
// the partition. It never exposes raw component values.
func (p Partition) String() string {
	if p.IsZero() {
		return "partition(unset)"
	}
	return fmt.Sprintf("partition(%016x)", p.hash)
}

// Partitioner assigns a partition to a key. Partition extraction runs in the
// caller goroutine before the request is enqueued, so it may read the caller
// context.
type Partitioner[K any] interface {
	// Partition returns the partition for key, or an error to fail the request
	// before it is enqueued.
	Partition(ctx context.Context, key K) (Partition, error)
}

// PartitionerFunc adapts a function to Partitioner.
type PartitionerFunc[K any] func(ctx context.Context, key K) (Partition, error)

// Partition calls the function.
func (f PartitionerFunc[K]) Partition(ctx context.Context, key K) (Partition, error) {
	return f(ctx, key)
}

// singlePartitioner assigns every key to SinglePartition.
type singlePartitioner[K any] struct{}

func (singlePartitioner[K]) Partition(context.Context, K) (Partition, error) {
	return SinglePartition(), nil
}

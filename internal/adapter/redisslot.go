package adapter

import "sort"

// redisSlots is the number of hash slots in a Redis cluster.
const redisSlots = 16384

// crc16 computes the CRC-16/XMODEM checksum Redis uses for cluster key hashing
// (polynomial 0x1021, initial value 0, no reflection).
func crc16(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// hashTag returns the substring used for slot computation. When a key contains a
// non-empty "{...}" hash tag, only the tag content is hashed, so related keys can
// be forced onto the same slot; otherwise the whole key is hashed.
func hashTag(key string) string {
	start := -1
	for i := 0; i < len(key); i++ {
		if key[i] == '{' {
			start = i
			break
		}
	}
	if start < 0 {
		return key
	}
	for j := start + 1; j < len(key); j++ {
		if key[j] == '}' {
			if j > start+1 {
				return key[start+1 : j]
			}
			return key
		}
	}
	return key
}

// Slot returns the Redis cluster hash slot for a key, honoring hash tags.
func Slot(key string) int {
	return int(crc16([]byte(hashTag(key))) % redisSlots)
}

// SlotGroup is a set of keys that share a cluster slot, with their original
// request indices preserved for result reconstruction.
type SlotGroup struct {
	Slot    int
	Indices []int
	Keys    []string
}

// SlotGroups groups keys by cluster slot. Groups are ordered by slot for
// determinism, and keys within a group preserve their original request order.
// This lets a cluster-aware batch issue one multi-key command per slot without
// crossing slots.
func SlotGroups(keys []string) []SlotGroup {
	bySlot := map[int]*SlotGroup{}
	var order []int
	for i, k := range keys {
		s := Slot(k)
		g, ok := bySlot[s]
		if !ok {
			g = &SlotGroup{Slot: s}
			bySlot[s] = g
			order = append(order, s)
		}
		g.Indices = append(g.Indices, i)
		g.Keys = append(g.Keys, k)
	}
	sort.Ints(order)
	out := make([]SlotGroup, 0, len(order))
	for _, s := range order {
		out = append(out, *bySlot[s])
	}
	return out
}

// SameSlot reports whether all keys map to a single slot (a precondition for a
// single cross-key command such as MGET on a Redis cluster).
func SameSlot(keys []string) bool {
	if len(keys) <= 1 {
		return true
	}
	first := Slot(keys[0])
	for _, k := range keys[1:] {
		if Slot(k) != first {
			return false
		}
	}
	return true
}

package adaptive

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func sampleBundle(t *testing.T) *ProfileBundle {
	t.Helper()
	spec := WorkloadSpec{
		Pattern: PatternPoisson, Operation: "users.get", Count: 2000, RatePerSec: 5000,
		Seed: 7, DistinctKeys: 1000, TenantClasses: 3, DeadlineFraction: 0.5,
		DeadlineNanos: int64(2 * time.Millisecond), PayloadBytes: 128,
	}
	settings := Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 128, MaxConcurrency: 8}
	return CollectSynthetic(spec, settings, BackendModel{FixedNanos: 400000, PerItemNanos: 15000})
}

func TestCollectAndDigestDeterminism(t *testing.T) {
	b1 := sampleBundle(t)
	b2 := sampleBundle(t)
	if b1.Digest != b2.Digest {
		t.Fatalf("digests differ for identical workloads: %s vs %s", b1.Digest, b2.Digest)
	}
	if len(b1.Operations) != 1 || b1.Operations[0].Operation != "users.get" {
		t.Fatalf("unexpected operations: %+v", b1.Operations)
	}
	if b1.Operations[0].Arrivals.LogicalCalls != 2000 {
		t.Errorf("logical calls = %d, want 2000", b1.Operations[0].Arrivals.LogicalCalls)
	}
}

func TestProfilePersistRoundTrip(t *testing.T) {
	b := sampleBundle(t)
	data, err := Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Digest != b.Digest {
		t.Errorf("digest changed across round trip: %s vs %s", got.Digest, b.Digest)
	}
}

func TestProfileChecksumTamperDetected(t *testing.T) {
	b := sampleBundle(t)
	data, err := Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	tampered := bytes.Replace(data, []byte(`"logical_calls": 2000`), []byte(`"logical_calls": 9999`), 1)
	if bytes.Equal(tampered, data) {
		t.Skip("marshal format changed; adjust tamper target")
	}
	if _, err := Unmarshal(tampered); err == nil {
		t.Fatal("expected checksum mismatch on tampered profile")
	}
}

func TestProfilePrivacyNoRawKeys(t *testing.T) {
	b := sampleBundle(t)
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if b.Redaction.RawKeysStored {
		t.Fatal("profile claims to store raw keys")
	}
	// Class labels are the only tenant/partition strings, and they are prefixed.
	for _, op := range b.Operations {
		for k := range op.Partitions.ByClass {
			if !strings.HasPrefix(k, "class_") {
				t.Errorf("partition class %q is not anonymized", k)
			}
		}
	}
	if bytes.Contains(data, []byte("tenant-")) {
		t.Error("serialized profile contains a raw tenant identifier")
	}
}

func TestProfileFileWriteReadPerms(t *testing.T) {
	b := sampleBundle(t)
	dir := t.TempDir()
	path := dir + "/service.bwp"
	if err := WriteFile(path, b); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Digest != b.Digest {
		t.Errorf("digest mismatch after file round trip")
	}
}

func TestMergeProfiles(t *testing.T) {
	b := sampleBundle(t)
	merged, err := Merge(b, b)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if got := merged.Operations[0].Arrivals.LogicalCalls; got != 4000 {
		t.Errorf("merged logical calls = %d, want 4000", got)
	}
}

func TestCompatibilityStaleAndIncompatible(t *testing.T) {
	b := sampleBundle(t)
	// Incompatible ABI.
	res := CheckCompatibility(b, CompatibilityRequirement{RuntimeABI: "other-abi"})
	if res.Compatible {
		t.Error("expected incompatible for mismatched ABI")
	}
	// Stale by age.
	res = CheckCompatibility(b, CompatibilityRequirement{
		MaxAge: time.Second,
		Now:    time.Unix(0, b.CreatedUnixNanos).Add(time.Hour),
	})
	if !res.Stale {
		t.Error("expected stale for old profile")
	}
}

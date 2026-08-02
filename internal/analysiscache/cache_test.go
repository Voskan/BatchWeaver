package analysiscache

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestKeyCoversCompatibilityDimensions(t *testing.T) {
	base := KeyInput{
		Workspace: "/workspace/a", ToolVersion: "v1", GoVersion: "go1.26",
		GOOS: "linux", GOARCH: "amd64", Tags: []string{"a", "b"}, Patterns: []string{"./..."},
		SourceDigest: "sha256:source", AnalysisSchema: "analysis-v1", ProofSchema: "proof-v1",
		TransformSchema: "transform-v1", StrategyVersion: "1", Reproducible: true,
	}
	want := Key(base)
	reordered := base
	reordered.Tags = []string{"b", "a"}
	if Key(reordered) != want {
		t.Fatal("tag ordering changed a semantic cache key")
	}
	mutations := []func(*KeyInput){
		func(k *KeyInput) { k.Workspace = "/workspace/b" },
		func(k *KeyInput) { k.ToolVersion = "v2" },
		func(k *KeyInput) { k.GoVersion = "go1.27" },
		func(k *KeyInput) { k.GOOS = "darwin" },
		func(k *KeyInput) { k.GOARCH = "arm64" },
		func(k *KeyInput) { k.CGOEnabled = true },
		func(k *KeyInput) { k.Tests = true },
		func(k *KeyInput) { k.Reproducible = false },
		func(k *KeyInput) { k.Tags = []string{"different"} },
		func(k *KeyInput) { k.Patterns = []string{"./cmd/..."} },
		func(k *KeyInput) { k.SourceDigest = "sha256:changed" },
		func(k *KeyInput) { k.AnalysisSchema = "analysis-v2" },
		func(k *KeyInput) { k.ProofSchema = "proof-v2" },
		func(k *KeyInput) { k.TransformSchema = "transform-v2" },
		func(k *KeyInput) { k.StrategyVersion = "2" },
	}
	for index, mutate := range mutations {
		candidate := base
		mutate(&candidate)
		if Key(candidate) == want {
			t.Errorf("dimension mutation %d did not invalidate the key", index)
		}
	}
}

func TestCacheBoundsDiskTier(t *testing.T) {
	dir := t.TempDir()
	cache, err := New(dir, Config{MaxEntries: 1, MaxBytes: 512, MaxDiskBytes: 700})
	if err != nil {
		t.Fatal(err)
	}
	for _, digest := range []string{"one", "two", "three"} {
		key := Key(KeyInput{Workspace: "w", SourceDigest: digest})
		if _, _, err := cache.GetOrCompute(context.Background(), key, func() ([]byte, error) {
			return make([]byte, 300), nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var diskBytes int64
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			t.Fatal(err)
		}
		diskBytes += info.Size()
	}
	if diskBytes > 700 || cache.Snapshot().Evictions == 0 {
		t.Fatalf("disk bytes=%d stats=%+v", diskBytes, cache.Snapshot())
	}
}

func TestCacheSingleFlightAndLRU(t *testing.T) {
	cache, err := New(t.TempDir(), Config{MaxEntries: 2, MaxBytes: 8, MaxDiskBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	key := Key(KeyInput{Workspace: "w", SourceDigest: "a"})
	var computes atomic.Int32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			value, _, err := cache.GetOrCompute(context.Background(), key, func() ([]byte, error) {
				computes.Add(1)
				time.Sleep(20 * time.Millisecond)
				return []byte("value"), nil
			})
			if err != nil || string(value) != "value" {
				t.Errorf("lookup = %q, %v", value, err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if computes.Load() != 1 {
		t.Fatalf("computes = %d, want 1", computes.Load())
	}
	for _, suffix := range []string{"b", "c"} {
		_, _, err := cache.GetOrCompute(context.Background(), Key(KeyInput{Workspace: "w", SourceDigest: suffix}), func() ([]byte, error) { return []byte("four"), nil })
		if err != nil {
			t.Fatal(err)
		}
	}
	stats := cache.Snapshot()
	if stats.Entries > 2 || stats.Bytes > 8 || stats.Evictions == 0 || stats.Hits < 15 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestCacheDiskRestartAndCorruption(t *testing.T) {
	dir := t.TempDir()
	key := Key(KeyInput{Workspace: "w", SourceDigest: "source"})
	first, _ := New(dir, Config{})
	if _, result, err := first.GetOrCompute(context.Background(), key, func() ([]byte, error) { return []byte("snapshot"), nil }); err != nil || result.Hit {
		t.Fatalf("initial result=%+v err=%v", result, err)
	}
	second, _ := New(dir, Config{})
	called := false
	value, result, err := second.GetOrCompute(context.Background(), key, func() ([]byte, error) { called = true; return nil, nil })
	if err != nil || called || !result.Hit || result.Source != "disk" || string(value) != "snapshot" {
		t.Fatalf("restart value=%q result=%+v called=%t err=%v", value, result, called, err)
	}
	if err := os.WriteFile(filepath.Join(dir, diskName(key)), []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	third, _ := New(dir, Config{})
	value, result, err = third.GetOrCompute(context.Background(), key, func() ([]byte, error) { return []byte("recomputed"), nil })
	if err != nil || result.Hit || string(value) != "recomputed" || third.Snapshot().Corruptions != 1 {
		t.Fatalf("corruption value=%q result=%+v stats=%+v err=%v", value, result, third.Snapshot(), err)
	}
}

func TestSourceDigestOverlayIsolationAndSymlinks(t *testing.T) {
	root := t.TempDir()
	writeCacheTestFile(t, filepath.Join(root, "go.mod"), "module example.com/cache\n")
	goFile := filepath.Join(root, "main.go")
	writeCacheTestFile(t, goFile, "package cache\n")
	base, err := SourceDigest(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := SourceDigest(root, map[string][]byte{goFile: []byte("package changed\n")})
	if err != nil || overlay == base {
		t.Fatalf("overlay digest=%q base=%q err=%v", overlay, base, err)
	}
	config := filepath.Join(root, "batchweaver.yaml")
	writeCacheTestFile(t, config, "version: 1\n")
	configured, err := SourceDigest(root, nil)
	if err != nil || configured == base {
		t.Fatalf("config digest=%q base=%q err=%v", configured, base, err)
	}
	writeCacheTestFile(t, config, "version: 2\n")
	reconfigured, err := SourceDigest(root, nil)
	if err != nil || reconfigured == configured {
		t.Fatalf("reconfigured digest=%q configured=%q err=%v", reconfigured, configured, err)
	}
	outside := filepath.Join(t.TempDir(), "outside.go")
	writeCacheTestFile(t, outside, "package outside\n")
	if _, err := SourceDigest(root, map[string][]byte{outside: []byte("x")}); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("outside overlay error = %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := SourceDigest(root, nil); !errors.Is(err, ErrUnsafePath) {
		t.Fatalf("symlink error = %v", err)
	}
}

func BenchmarkCacheMemoryHit(b *testing.B) {
	cache, _ := New(b.TempDir(), Config{})
	key := Key(KeyInput{Workspace: "benchmark", SourceDigest: "source"})
	_, _, _ = cache.GetOrCompute(context.Background(), key, func() ([]byte, error) { return make([]byte, 4096), nil })
	b.ResetTimer()
	for range b.N {
		_, _, _ = cache.GetOrCompute(context.Background(), key, func() ([]byte, error) { b.Fatal("unexpected compute"); return nil, nil })
	}
}

func writeCacheTestFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}

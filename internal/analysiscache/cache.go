// Package analysiscache implements the bounded, content-addressed cache used by
// the local workspace daemon. It stores only serialized analysis snapshots;
// source and overlay bytes participate in keys but are never persisted.
package analysiscache

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// SchemaVersion identifies both key and disk-envelope compatibility.
const SchemaVersion = "batchweaver.analysis-cache/v1alpha1"

// ErrUnsafePath marks a workspace or overlay path that escapes isolation.
var ErrUnsafePath = errors.New("analysis cache: unsafe path")

// Config bounds memory and disk use. Zero fields receive conservative defaults.
type Config struct {
	MaxEntries   int
	MaxBytes     int64
	MaxDiskBytes int64
}

func normalizedConfig(config Config) Config {
	if config.MaxEntries <= 0 {
		config.MaxEntries = 32
	}
	if config.MaxBytes <= 0 {
		config.MaxBytes = 128 << 20
	}
	if config.MaxDiskBytes <= 0 {
		config.MaxDiskBytes = 512 << 20
	}
	return config
}

// KeyInput contains every compatibility dimension that can change analysis.
type KeyInput struct {
	Workspace       string
	ToolVersion     string
	GoVersion       string
	GOOS            string
	GOARCH          string
	CGOEnabled      bool
	Tests           bool
	Reproducible    bool
	Tags            []string
	Patterns        []string
	SourceDigest    string
	AnalysisSchema  string
	ProofSchema     string
	TransformSchema string
	StrategyVersion string
}

// Key returns a stable SHA-256 cache identity without exposing workspace paths.
func Key(input KeyInput) string {
	workspace := sha256.Sum256([]byte(input.Workspace))
	tags := append([]string(nil), input.Tags...)
	patterns := append([]string(nil), input.Patterns...)
	sort.Strings(tags)
	sort.Strings(patterns)
	parts := []string{
		SchemaVersion, hex.EncodeToString(workspace[:]), input.ToolVersion,
		input.GoVersion, input.GOOS, input.GOARCH, fmt.Sprint(input.CGOEnabled),
		fmt.Sprint(input.Tests), fmt.Sprint(input.Reproducible), strings.Join(tags, ","), strings.Join(patterns, "\x00"),
		input.SourceDigest, input.AnalysisSchema, input.ProofSchema,
		input.TransformSchema, input.StrategyVersion,
	}
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

// Stats is privacy-safe cache observability. It contains counts and byte sizes,
// never keys, paths, source, payload values, or tenant identities.
type Stats struct {
	Hits        uint64 `json:"hits"`
	Misses      uint64 `json:"misses"`
	DiskHits    uint64 `json:"disk_hits"`
	Evictions   uint64 `json:"evictions"`
	Corruptions uint64 `json:"corruptions"`
	Entries     int    `json:"entries"`
	Bytes       int64  `json:"bytes"`
}

// Result describes one lookup without revealing the content-addressed key.
type Result struct {
	Hit    bool   `json:"hit"`
	Source string `json:"source"` // memory, disk, compute, or shared
}

type entry struct {
	key   string
	value []byte
	size  int64
}

type call struct {
	done  chan struct{}
	value []byte
	err   error
}

// Cache is a concurrent single-flight LRU with an optional bounded disk tier.
type Cache struct {
	mu       sync.Mutex
	config   Config
	dir      string
	items    map[string]*list.Element
	lru      *list.List
	inflight map[string]*call
	bytes    int64
	stats    Stats
}

// New creates a cache. dir must already be scoped to one validated workspace.
func New(dir string, config Config) (*Cache, error) {
	if dir == "" {
		return nil, fmt.Errorf("analysis cache: disk directory is required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, err
	}
	return &Cache{config: normalizedConfig(config), dir: dir, items: map[string]*list.Element{}, lru: list.New(), inflight: map[string]*call{}}, nil
}

// GetOrCompute returns a defensive copy. Concurrent misses for the same key run
// compute exactly once; canceled waiters return without blocking the leader.
func (c *Cache) GetOrCompute(ctx context.Context, key string, compute func() ([]byte, error)) ([]byte, Result, error) {
	if !validKey(key) {
		return nil, Result{}, fmt.Errorf("analysis cache: invalid key")
	}
	c.mu.Lock()
	if elem := c.items[key]; elem != nil {
		c.lru.MoveToFront(elem)
		c.stats.Hits++
		value := clone(elem.Value.(*entry).value)
		c.mu.Unlock()
		return value, Result{Hit: true, Source: "memory"}, nil
	}
	if active := c.inflight[key]; active != nil {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, Result{}, ctx.Err()
		case <-active.done:
			if active.err != nil {
				return nil, Result{}, active.err
			}
			c.mu.Lock()
			c.stats.Hits++
			c.mu.Unlock()
			return clone(active.value), Result{Hit: true, Source: "shared"}, nil
		}
	}
	active := &call{done: make(chan struct{})}
	c.inflight[key] = active
	c.mu.Unlock()

	value, err := c.loadDisk(key)
	source := "disk"
	if err == nil && value != nil {
		c.mu.Lock()
		c.stats.Hits++
		c.stats.DiskHits++
		c.insertLocked(key, value)
		c.mu.Unlock()
	} else {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		value, err = compute()
		source = "compute"
		if err == nil {
			c.mu.Lock()
			c.insertLocked(key, value)
			c.mu.Unlock()
			_ = c.writeDisk(key, value) // a disk-tier failure never discards a valid analysis
		}
	}

	c.mu.Lock()
	active.value, active.err = clone(value), err
	delete(c.inflight, key)
	close(active.done)
	c.mu.Unlock()
	if err != nil {
		return nil, Result{}, err
	}
	return clone(value), Result{Hit: source != "compute", Source: source}, nil
}

// Snapshot returns current counters and in-memory occupancy.
func (c *Cache) Snapshot() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	stats := c.stats
	stats.Entries, stats.Bytes = len(c.items), c.bytes
	return stats
}

func (c *Cache) insertLocked(key string, value []byte) {
	if int64(len(value)) > c.config.MaxBytes {
		return
	}
	if old := c.items[key]; old != nil {
		item := old.Value.(*entry)
		c.bytes -= item.size
		item.value, item.size = clone(value), int64(len(value))
		c.bytes += item.size
		c.lru.MoveToFront(old)
		return
	}
	item := &entry{key: key, value: clone(value), size: int64(len(value))}
	c.items[key] = c.lru.PushFront(item)
	c.bytes += item.size
	for len(c.items) > c.config.MaxEntries || c.bytes > c.config.MaxBytes {
		last := c.lru.Back()
		if last == nil {
			break
		}
		victim := last.Value.(*entry)
		delete(c.items, victim.key)
		c.bytes -= victim.size
		c.lru.Remove(last)
		c.stats.Evictions++
	}
}

type envelope struct {
	Schema  string `json:"schema"`
	Key     string `json:"key"`
	Digest  string `json:"digest"`
	Payload []byte `json:"payload"`
}

func (c *Cache) loadDisk(key string) ([]byte, error) {
	path := filepath.Join(c.dir, diskName(key))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var stored envelope
	if json.Unmarshal(data, &stored) != nil || stored.Schema != SchemaVersion || stored.Key != key || stored.Digest != payloadDigest(stored.Payload) {
		_ = os.Remove(path)
		c.mu.Lock()
		c.stats.Corruptions++
		c.mu.Unlock()
		return nil, nil //nolint:nilerr // invalid cache data is a miss, not an analysis failure
	}
	now := time.Now()
	_ = os.Chtimes(path, now, now)
	return stored.Payload, nil
}

func (c *Cache) writeDisk(key string, value []byte) error {
	stored := envelope{Schema: SchemaVersion, Key: key, Digest: payloadDigest(value), Payload: value}
	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(c.dir, ".entry-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, filepath.Join(c.dir, diskName(key))); err != nil {
		return err
	}
	return c.enforceDiskLimit()
}

func (c *Cache) enforceDiskLimit() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	type diskEntry struct {
		path string
		size int64
		mod  time.Time
	}
	var files []diskEntry
	var total int64
	for _, item := range entries {
		if item.IsDir() || !strings.HasSuffix(item.Name(), ".json") {
			continue
		}
		info, err := item.Info()
		if err != nil {
			continue
		}
		files = append(files, diskEntry{filepath.Join(c.dir, item.Name()), info.Size(), info.ModTime()})
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
	for _, file := range files {
		if total <= c.config.MaxDiskBytes {
			break
		}
		if err := os.Remove(file.path); err == nil {
			total -= file.size
			c.mu.Lock()
			c.stats.Evictions++
			c.mu.Unlock()
		}
	}
	return nil
}

func validKey(key string) bool {
	if !strings.HasPrefix(key, "sha256:") || len(key) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(key, "sha256:"))
	return err == nil
}

func diskName(key string) string { return strings.TrimPrefix(key, "sha256:") + ".json" }
func payloadDigest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func clone(value []byte) []byte { return append([]byte(nil), value...) }

// SourceDigest hashes relevant on-disk workspace inputs plus authoritative
// overlay bytes. Symlinks are rejected so one workspace cannot alias another.
func SourceDigest(root string, overlay map[string][]byte) (string, error) {
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	canonicalRoot, err = filepath.Abs(canonicalRoot)
	if err != nil {
		return "", err
	}
	overrides := make(map[string][]byte, len(overlay))
	for path, content := range overlay {
		inputAbs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(inputAbs))
		if err != nil {
			return "", fmt.Errorf("%w: resolve overlay parent: %w", ErrUnsafePath, err)
		}
		abs := filepath.Join(parent, filepath.Base(inputAbs))
		rel, err := filepath.Rel(canonicalRoot, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("%w: overlay escapes workspace", ErrUnsafePath)
		}
		parentRel, err := filepath.Rel(canonicalRoot, parent)
		if err != nil || parentRel == ".." || strings.HasPrefix(parentRel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("%w: overlay parent escapes workspace", ErrUnsafePath)
		}
		if info, err := os.Lstat(abs); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: overlay target is a symlink", ErrUnsafePath)
		}
		overrides[filepath.ToSlash(rel)] = content
	}
	type source struct {
		path string
		data []byte
	}
	var sources []source
	err = filepath.WalkDir(canonicalRoot, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == canonicalRoot {
			return nil
		}
		name := item.Name()
		if item.IsDir() && (name == ".git" || name == ".batchweaver" || name == "vendor" || name == "node_modules") {
			return filepath.SkipDir
		}
		if item.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", ErrUnsafePath, name)
		}
		if item.IsDir() || !relevantSource(name) {
			return nil
		}
		rel, err := filepath.Rel(canonicalRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if _, replaced := overrides[rel]; replaced {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sources = append(sources, source{rel, data})
		return nil
	})
	if err != nil {
		return "", err
	}
	for path, data := range overrides {
		sources = append(sources, source{path, data})
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].path < sources[j].path })
	hash := sha256.New()
	for _, item := range sources {
		_, _ = hash.Write([]byte(item.path))
		_, _ = hash.Write([]byte{0})
		sum := sha256.Sum256(item.data)
		_, _ = hash.Write(sum[:])
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func relevantSource(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".go" || ext == ".mod" || ext == ".sum" || ext == ".work" || ext == ".yaml" || ext == ".yml" || ext == ".json"
}

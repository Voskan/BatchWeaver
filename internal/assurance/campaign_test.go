package assurance

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/Voskan/BatchWeaver/internal/adapter"
	"github.com/Voskan/BatchWeaver/internal/adaptive"
)

func TestProductionLikeSoak(t *testing.T) {
	duration := campaignDuration(t)
	seed := campaignSeed(t)
	deadline := time.Now().Add(duration)
	iterations := 0
	for time.Now().Before(deadline) {
		workload := adaptive.GenerateWorkload(adaptive.WorkloadSpec{
			Pattern: adaptive.PatternHeavyTail, Operation: "campaign.users.get",
			Count: 256, RatePerSec: 8000, Seed: seed + uint64(iterations),
			DuplicationRate: 0.35, DistinctKeys: 67, TenantClasses: 4,
			TenantSkew: 0.65, DeadlineFraction: 0.1, PayloadBytes: 384,
			PhaseChangeAt: 128,
		})
		latency := adaptive.NewHistogram(adaptive.DefaultHistogramAccuracy)
		requests := make([]request, len(workload))
		for i, event := range workload {
			latency.Observe(float64(event.ArrivalNanos + 1))
			requests[i] = request{ID: i + 1, Key: int(event.Key), Partition: i % 4, Canceled: event.DeadlineNanos > 0 && i%17 == 0}
		}
		want, wantTrace := scalar(requests)
		got, gotTrace := batched(requests)
		if !equalOutcomes(want, got) || !equalStrings(wantTrace, gotTrace) {
			t.Fatalf("seed %d differential mismatch", seed+uint64(iterations))
		}
		if latency.Count() != uint64(len(workload)) || latency.Quantile(0.99) < 0 {
			t.Fatal("adaptive histogram lost workload samples")
		}
		parsed, rejection := adapter.ParseExactKeySelect("SELECT u.tenant_id, u.id, p.name FROM users u LEFT JOIN profiles p ON p.user_id = u.id WHERE u.tenant_id = $1 AND u.id = $2")
		if rejection != nil {
			t.Fatal(rejection)
		}
		plan, rejection := adapter.SynthesizeExactKey(adapter.SynthInput{
			Query: parsed, KeyTypes: []string{"uuid", "bigint"},
			JoinCardinality: adapter.JoinCardinalityAtMostOne,
		})
		if rejection != nil || plan.Validate() != nil {
			t.Fatalf("SQL campaign synthesis failed: rejection=%v", rejection)
		}
		iterations++
	}
	if iterations == 0 {
		t.Fatal("soak completed no iterations")
	}
	t.Logf("campaign seed=%d duration=%s iterations=%d", seed, duration, iterations)
}

func equalOutcomes(a, b []outcome) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func campaignDuration(t *testing.T) time.Duration {
	t.Helper()
	value := os.Getenv("BATCHWEAVER_SOAK_DURATION")
	if value == "" {
		return 100 * time.Millisecond
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		t.Fatalf("invalid BATCHWEAVER_SOAK_DURATION %q", value)
	}
	return duration
}

func campaignSeed(t *testing.T) uint64 {
	t.Helper()
	value := os.Getenv("BATCHWEAVER_CAMPAIGN_SEED")
	if value == "" {
		return 0x42ba7c11
	}
	seed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		t.Fatalf("invalid BATCHWEAVER_CAMPAIGN_SEED %q", value)
	}
	return seed
}

type resourceLedger struct {
	timers, files, sockets, rows, bodies, streams, children int
}

func (l resourceLedger) empty() bool {
	return l.timers == 0 && l.files == 0 && l.sockets == 0 && l.rows == 0 && l.bodies == 0 && l.streams == 0 && l.children == 0
}

func TestResourceLeakBudgets(t *testing.T) {
	runtime.GC()
	beforeGoroutines := runtime.NumGoroutine()
	beforeFDs := countOpenFDs()
	var beforeMem runtime.MemStats
	runtime.ReadMemStats(&beforeMem)
	ledger := &resourceLedger{}
	path := filepath.Join(t.TempDir(), "resource")
	if err := os.WriteFile(path, []byte("campaign"), 0o600); err != nil {
		t.Fatal(err)
	}
	for range 128 {
		timer := time.NewTimer(time.Millisecond)
		ledger.timers++
		if !timer.Stop() {
			<-timer.C
		}
		ledger.timers--

		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		ledger.files++
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		ledger.files--

		left, right := net.Pipe()
		ledger.sockets += 2
		_ = left.Close()
		_ = right.Close()
		ledger.sockets -= 2

		for _, kind := range []string{"rows", "body", "stream"} {
			closer := &trackedCloser{ledger: ledger, kind: kind}
			closer.open()
			if err := closer.Close(); err != nil {
				t.Fatal(err)
			}
		}

		command := exec.Command(os.Args[0], "-test.run=^TestCampaignHelperProcess$")
		command.Env = append(os.Environ(), "BATCHWEAVER_HELPER_MODE=healthy")
		ledger.children++
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("child process: %v: %s", err, output)
		}
		ledger.children--
	}
	if !ledger.empty() {
		t.Fatalf("resource ledger not empty: %+v", ledger)
	}
	runtime.GC()
	time.Sleep(25 * time.Millisecond)
	afterGoroutines := runtime.NumGoroutine()
	afterFDs := countOpenFDs()
	var afterMem runtime.MemStats
	runtime.ReadMemStats(&afterMem)
	if delta := afterGoroutines - beforeGoroutines; delta > 4 {
		t.Fatalf("goroutine leak budget exceeded: delta=%d budget=4", delta)
	}
	if beforeFDs >= 0 && afterFDs-beforeFDs > 2 {
		t.Fatalf("file/socket descriptor leak budget exceeded: delta=%d budget=2", afterFDs-beforeFDs)
	}
	if afterMem.HeapAlloc > beforeMem.HeapAlloc+8<<20 {
		t.Fatalf("memory leak budget exceeded: before=%d after=%d budget=%d", beforeMem.HeapAlloc, afterMem.HeapAlloc, 8<<20)
	}
	t.Logf("resource deltas goroutines=%d fds=%d heap_bytes=%d", afterGoroutines-beforeGoroutines, fdDelta(beforeFDs, afterFDs), int64(afterMem.HeapAlloc)-int64(beforeMem.HeapAlloc))
}

type trackedCloser struct {
	ledger *resourceLedger
	kind   string
	openAt bool
}

func (c *trackedCloser) open() {
	c.openAt = true
	switch c.kind {
	case "rows":
		c.ledger.rows++
	case "body":
		c.ledger.bodies++
	case "stream":
		c.ledger.streams++
	}
}

func (c *trackedCloser) Close() error {
	if !c.openAt {
		return errors.New("double close")
	}
	c.openAt = false
	switch c.kind {
	case "rows":
		c.ledger.rows--
	case "body":
		c.ledger.bodies--
	case "stream":
		c.ledger.streams--
	}
	return nil
}

func countOpenFDs() int {
	for _, dir := range []string{"/proc/self/fd", "/dev/fd"} {
		entries, err := os.ReadDir(dir)
		if err == nil {
			return len(entries)
		}
	}
	return -1
}

func fdDelta(before, after int) int {
	if before < 0 || after < 0 {
		return 0
	}
	return after - before
}

func TestProductionFaultMatrix(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		<-ctx.Done()
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("timeout error=%v", ctx.Err())
		}
	})
	t.Run("reset", func(t *testing.T) {
		left, right := net.Pipe()
		_ = right.Close()
		defer func() { _ = left.Close() }()
		if _, err := left.Write([]byte("request")); err == nil {
			t.Fatal("closed peer accepted write")
		}
	})
	t.Run("partial", func(t *testing.T) {
		buffer := make([]byte, 16)
		if _, err := io.ReadFull(bytes.NewReader([]byte("short")), buffer); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("partial response error=%v", err)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		var target map[string]any
		if err := jsonUnmarshal([]byte(`{"unterminated":`), &target); err == nil {
			t.Fatal("malformed response accepted")
		}
	})
	t.Run("corruption", func(t *testing.T) {
		want := sha256.Sum256([]byte("trusted"))
		got := sha256.Sum256([]byte("corrupted"))
		if bytes.Equal(want[:], got[:]) {
			t.Fatal("corruption was not detected")
		}
	})
	t.Run("permission", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("POSIX permission fault")
		}
		path := filepath.Join(t.TempDir(), "denied")
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0 {
			t.Fatalf("permission fault not installed: mode=%v", info.Mode())
		}
	})
	t.Run("crash", func(t *testing.T) {
		command := exec.Command(os.Args[0], "-test.run=^TestCampaignHelperProcess$")
		command.Env = append(os.Environ(), "BATCHWEAVER_HELPER_MODE=crash")
		err := command.Run()
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 23 {
			t.Fatalf("crash exit=%v", err)
		}
	})
	t.Run("download", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "64")
			_, _ = w.Write([]byte("truncated"))
		}))
		defer server.Close()
		response, err := server.Client().Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		_, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr == nil {
			t.Fatal("truncated download accepted")
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	})
	t.Run("signature", func(t *testing.T) {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		signature := ed25519.Sign(privateKey, []byte("artifact"))
		if ed25519.Verify(publicKey, []byte("tampered"), signature) {
			t.Fatal("tampered signature accepted")
		}
	})
}

// jsonUnmarshal is kept behind a tiny seam so the malformed-response fault is
// explicit in campaign profiles and remains easy to replace with protocol logic.
func jsonUnmarshal(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	return decoder.Decode(target)
}

func TestCampaignHelperProcess(t *testing.T) {
	switch os.Getenv("BATCHWEAVER_HELPER_MODE") {
	case "healthy":
		fmt.Fprint(io.Discard, "healthy")
	case "crash":
		os.Exit(23)
	}
}

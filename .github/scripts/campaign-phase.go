// Command campaign-phase executes one immutable campaign command and writes a
// machine-readable evidence envelope even when the command fails.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
)

const evidenceSchema = "batchweaver.campaign-phase/v1alpha1"

type resourceSnapshot struct {
	Goroutines int    `json:"goroutines"`
	HeapBytes  uint64 `json:"heap_bytes"`
	OpenFDs    int    `json:"open_fds"`
	ChildRSSKB int64  `json:"child_max_rss_kb"`
}

type resourceDelta struct {
	Goroutines int   `json:"goroutines"`
	HeapBytes  int64 `json:"heap_bytes"`
	OpenFDs    int   `json:"open_fds"`
	ChildRSSKB int64 `json:"child_max_rss_kb"`
}

type evidence struct {
	Schema       string            `json:"schema"`
	Name         string            `json:"name"`
	Category     string            `json:"category"`
	Commit       string            `json:"commit"`
	RunURL       string            `json:"run_url"`
	Command      []string          `json:"command"`
	Environment  map[string]string `json:"environment"`
	Seeds        []string          `json:"seeds"`
	CorpusDigest string            `json:"corpus_digest,omitempty"`
	StartedAt    string            `json:"started_at"`
	FinishedAt   string            `json:"finished_at"`
	DurationMS   int64             `json:"duration_ms"`
	Resources    resourceDelta     `json:"resource_deltas"`
	Status       string            `json:"status"`
	ExitCode     int               `json:"exit_code"`
	Failure      string            `json:"failure,omitempty"`
	Log          string            `json:"log"`
}

func main() {
	var output, name, category string
	flag.StringVar(&output, "output", "", "evidence JSON path")
	flag.StringVar(&name, "name", "", "phase name")
	flag.StringVar(&category, "category", "", "phase category")
	flag.Parse()
	command := flag.Args()
	if output == "" || name == "" || category == "" || len(command) == 0 {
		fmt.Fprintln(os.Stderr, "usage: campaign-phase --output FILE --name NAME --category CATEGORY COMMAND [ARG...]")
		os.Exit(2)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	logPath := strings.TrimSuffix(output, filepath.Ext(output)) + ".log"
	logFile, err := os.Create(logPath) //nolint:gosec // path is an explicit CI output.
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	start := time.Now().UTC()
	before := snapshot()
	cmd := exec.Command(command[0], command[1:]...) //nolint:gosec // fixed workflow command, recorded verbatim.
	cmd.Stdout = io.MultiWriter(os.Stdout, logFile)
	cmd.Stderr = io.MultiWriter(os.Stderr, logFile)
	runErr := cmd.Run()
	_ = logFile.Close()
	after := snapshot()
	finish := time.Now().UTC()
	exitCode := 0
	status := "pass"
	failure := ""
	if runErr != nil {
		status, failure = "fail", runErr.Error()
		exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	record := evidence{
		Schema: evidenceSchema, Name: name, Category: category,
		Commit: commit(), RunURL: runURL(), Command: command,
		Environment: environment(), Seeds: seeds(command),
		CorpusDigest: corpusDigest(os.Getenv("BATCHWEAVER_FUZZ_CORPUS")),
		StartedAt:    start.Format(time.RFC3339Nano), FinishedAt: finish.Format(time.RFC3339Nano),
		DurationMS: finish.Sub(start).Milliseconds(), Resources: delta(before, after),
		Status: status, ExitCode: exitCode, Failure: failure, Log: filepath.Base(logPath),
	}
	data, marshalErr := json.MarshalIndent(record, "", "  ")
	if marshalErr == nil {
		marshalErr = os.WriteFile(output, append(data, '\n'), 0o644)
	}
	if marshalErr != nil {
		fmt.Fprintln(os.Stderr, marshalErr)
		os.Exit(2)
	}
	if runErr != nil {
		os.Exit(exitCode)
	}
}

func environment() map[string]string {
	values := map[string]string{
		"go": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH,
		"github_event": os.Getenv("GITHUB_EVENT_NAME"), "github_ref": os.Getenv("GITHUB_REF"),
		"runner_os": os.Getenv("RUNNER_OS"), "runner_arch": os.Getenv("RUNNER_ARCH"),
	}
	if output, err := exec.Command("go", "env", "GOTOOLCHAIN", "CGO_ENABLED").Output(); err == nil { //nolint:gosec
		parts := strings.Fields(string(output))
		if len(parts) == 2 {
			values["gotoolchain"], values["cgo_enabled"] = parts[0], parts[1]
		}
	}
	return values
}

func commit() string {
	if value := os.Getenv("GITHUB_SHA"); value != "" {
		return value
	}
	output, err := exec.Command("git", "rev-parse", "HEAD").Output() //nolint:gosec
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func runURL() string {
	server, repository, id := os.Getenv("GITHUB_SERVER_URL"), os.Getenv("GITHUB_REPOSITORY"), os.Getenv("GITHUB_RUN_ID")
	if server == "" || repository == "" || id == "" {
		return "local"
	}
	return server + "/" + repository + "/actions/runs/" + id
}

func seeds(command []string) []string {
	result := []string{"campaign:" + valueOr(os.Getenv("BATCHWEAVER_CAMPAIGN_SEED"), "1119513617")}
	for _, arg := range command {
		if strings.Contains(arg, "-fuzz") {
			result = append(result, "go-fuzz:"+arg)
		}
	}
	sort.Strings(result)
	return result
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func snapshot() resourceSnapshot {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	return resourceSnapshot{
		Goroutines: runtime.NumGoroutine(), HeapBytes: memory.HeapAlloc,
		OpenFDs: openFDs(), ChildRSSKB: childRSSKB(),
	}
}

func delta(before, after resourceSnapshot) resourceDelta {
	fd := 0
	if before.OpenFDs >= 0 && after.OpenFDs >= 0 {
		fd = after.OpenFDs - before.OpenFDs
	}
	return resourceDelta{
		Goroutines: after.Goroutines - before.Goroutines,
		HeapBytes:  int64(after.HeapBytes) - int64(before.HeapBytes), OpenFDs: fd,
		ChildRSSKB: after.ChildRSSKB - before.ChildRSSKB,
	}
}

func openFDs() int {
	for _, dir := range []string{"/proc/self/fd", "/dev/fd"} {
		entries, err := os.ReadDir(dir)
		if err == nil {
			return len(entries)
		}
	}
	return -1
}

func childRSSKB() int64 {
	var usage syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_CHILDREN, &usage) != nil {
		return 0
	}
	return usage.Maxrss
}

func corpusDigest(root string) string {
	if root == "" {
		return ""
	}
	hash := sha256.New()
	var paths []string
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	for _, path := range paths {
		rel, _ := filepath.Rel(root, path)
		_, _ = io.WriteString(hash, filepath.ToSlash(rel))
		data, err := os.ReadFile(path)
		if err == nil {
			_, _ = hash.Write(data)
		}
	}
	if len(paths) == 0 {
		return ""
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

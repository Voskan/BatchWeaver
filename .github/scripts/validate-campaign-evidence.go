// Command validate-campaign-evidence rejects incomplete, failed, or mixed-commit
// campaign artifacts and emits one combined hosted-evidence document.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const evidenceSchema = "batchweaver.campaign-phase/v1alpha1"

type phase struct {
	Schema      string            `json:"schema"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Commit      string            `json:"commit"`
	RunURL      string            `json:"run_url"`
	Command     []string          `json:"command"`
	Environment map[string]string `json:"environment"`
	Seeds       []string          `json:"seeds"`
	StartedAt   string            `json:"started_at"`
	FinishedAt  string            `json:"finished_at"`
	DurationMS  int64             `json:"duration_ms"`
	Status      string            `json:"status"`
	ExitCode    int               `json:"exit_code"`
	Failure     string            `json:"failure,omitempty"`
}

type campaign struct {
	Schema     string  `json:"schema"`
	Commit     string  `json:"commit"`
	RunURL     string  `json:"run_url"`
	Conclusion string  `json:"conclusion"`
	Phases     []phase `json:"phases"`
}

func main() {
	var root, commit, expected, output string
	flag.StringVar(&root, "root", "", "downloaded artifact root")
	flag.StringVar(&commit, "commit", "", "expected commit")
	flag.StringVar(&expected, "expected", "", "comma-separated categories")
	flag.StringVar(&output, "output", "campaign.json", "combined evidence path")
	flag.Parse()
	if root == "" || commit == "" || expected == "" {
		fatalf("root, commit, and expected are required")
	}
	var phases []phase
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var record phase
		if json.Unmarshal(data, &record) != nil || record.Schema != evidenceSchema {
			return nil
		}
		phases = append(phases, record)
		return nil
	})
	if err != nil {
		fatalf("walk evidence: %v", err)
	}
	if len(phases) == 0 {
		fatalf("no campaign phase evidence found")
	}
	wanted := map[string]bool{}
	for _, category := range strings.Split(expected, ",") {
		wanted[strings.TrimSpace(category)] = false
	}
	runURL := phases[0].RunURL
	sha := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for _, record := range phases {
		if record.Commit != commit || !sha.MatchString(record.Commit) {
			fatalf("phase %s commit %q does not match %q", record.Name, record.Commit, commit)
		}
		if record.RunURL == "" || record.RunURL != runURL {
			fatalf("phase %s has inconsistent run URL", record.Name)
		}
		if record.Name == "" || record.Category == "" || len(record.Command) == 0 || len(record.Environment) == 0 || len(record.Seeds) == 0 || record.StartedAt == "" || record.FinishedAt == "" || record.DurationMS < 0 {
			fatalf("phase %s has incomplete evidence", record.Name)
		}
		if record.Status != "pass" || record.ExitCode != 0 || record.Failure != "" {
			fatalf("phase %s failed: exit=%d failure=%s", record.Name, record.ExitCode, record.Failure)
		}
		if _, ok := wanted[record.Category]; ok {
			wanted[record.Category] = true
		}
	}
	for category, present := range wanted {
		if !present {
			fatalf("required category %q has no passing evidence", category)
		}
	}
	sort.Slice(phases, func(i, j int) bool { return phases[i].Name < phases[j].Name })
	combined := campaign{
		Schema: "batchweaver.campaign/v1alpha1", Commit: commit,
		RunURL: runURL, Conclusion: "pass", Phases: phases,
	}
	data, err := json.MarshalIndent(combined, "", "  ")
	if err != nil {
		fatalf("marshal combined evidence: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fatalf("create output directory: %v", err)
	}
	if err := os.WriteFile(output, append(data, '\n'), 0o644); err != nil {
		fatalf("write combined evidence: %v", err)
	}
	fmt.Printf("validated %d passing phases for %s\n", len(phases), commit)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

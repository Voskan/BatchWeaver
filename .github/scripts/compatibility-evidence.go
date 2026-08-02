// Command compatibility-evidence writes one hosted compatibility result.
// It lives below .github so it is not part of the shipped Go module packages.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type evidence struct {
	Schema      string `json:"schema"`
	Commit      string `json:"commit"`
	RunURL      string `json:"run_url"`
	WorkflowRef string `json:"workflow_ref"`
	RunnerOS    string `json:"runner_os"`
	RunnerArch  string `json:"runner_arch"`
	GoVersion   string `json:"go_version"`
	Dimension   string `json:"dimension"`
	Value       string `json:"value"`
	Command     string `json:"command"`
	Status      string `json:"status"`
	GeneratedAt string `json:"generated_at"`
}

func main() {
	output := flag.String("output", "", "output JSON path")
	dimension := flag.String("dimension", "", "matrix dimension")
	value := flag.String("value", "", "tested value")
	command := flag.String("command", "", "executed verification")
	status := flag.String("status", "", "GitHub job status")
	flag.Parse()
	if *output == "" || *dimension == "" || *value == "" || *command == "" || *status == "" {
		fmt.Fprintln(os.Stderr, "compatibility-evidence: every flag is required")
		os.Exit(2)
	}
	runURL := os.Getenv("GITHUB_SERVER_URL") + "/" + os.Getenv("GITHUB_REPOSITORY") + "/actions/runs/" + os.Getenv("GITHUB_RUN_ID")
	doc := evidence{
		Schema: "batchweaver.compatibility-evidence/v1alpha1", Commit: os.Getenv("GITHUB_SHA"),
		RunURL: runURL, WorkflowRef: os.Getenv("GITHUB_WORKFLOW_REF"), RunnerOS: os.Getenv("RUNNER_OS"),
		RunnerArch: os.Getenv("RUNNER_ARCH"), GoVersion: runtime.Version(), Dimension: *dimension,
		Value: *value, Command: *command, Status: *status, GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*output, append(data, '\n'), 0o644); err != nil {
		panic(err)
	}
}

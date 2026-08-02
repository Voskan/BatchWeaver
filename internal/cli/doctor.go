package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Voskan/BatchWeaver/internal/buildinfo"
)

type doctorReport struct {
	Schema        string         `json:"schema"`
	Build         buildinfo.Info `json:"build"`
	CPU           int            `json:"logical_cpu_count"`
	Privacy       string         `json:"privacy"`
	Included      []string       `json:"included"`
	Excluded      []string       `json:"excluded"`
	NetworkAccess string         `json:"network_access"`
}

func newDoctorCommand() *Command {
	return &Command{Name: "doctor", Summary: "Report a privacy-safe environment summary or support bundle", Usage: "doctor [--json] [--bundle PATH]", Run: runDoctor}
}

func runDoctor(_ context.Context, app *App, args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	jsonOut := fs.Bool("json", false, "emit JSON")
	bundle := fs.String("bundle", "", "write a privacy-safe .tar.gz support bundle")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage, Message: ""}
	}
	report := doctorReport{
		Schema:        "batchweaver.doctor/v1alpha1",
		Build:         buildinfo.Get(),
		CPU:           runtime.NumCPU(),
		Privacy:       "allowlist-only; no source, environment values, credentials, request data, or local paths",
		Included:      []string{"version", "commit", "Go version", "OS/architecture", "runtime ABI", "schema versions", "logical CPU count"},
		Excluded:      []string{"source", "configuration values", "environment values", "keys", "tokens", "URLs", "headers", "SQL", "GraphQL variables", "metadata", "tenant identifiers", "logs", "user and home paths"},
		NetworkAccess: "none",
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if *bundle != "" {
		if err := writeDoctorBundle(*bundle, data); err != nil {
			return err
		}
		fmt.Fprintf(app.Stdout(), "Privacy-safe support bundle written: %s\n", *bundle)
		return nil
	}
	if *jsonOut {
		_, err = app.Stdout().Write(data)
		return err
	}
	fmt.Fprintf(app.Stdout(), "BatchWeaver doctor\nVersion: %s\nChannel: %s\nGo: %s\nPlatform: %s\nRuntime ABI: %s\nPrivacy: %s\nNetwork access: none\n", report.Build.Version, report.Build.ReleaseChannel, report.Build.GoVersion, report.Build.Platform(), report.Build.RuntimeABI, report.Privacy)
	return nil
}

func writeDoctorBundle(path string, data []byte) error {
	if filepath.Ext(path) != ".gz" {
		return fmt.Errorf("doctor bundle path must end in .tar.gz")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	gz, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		return err
	}
	gz.ModTime = time.Unix(0, 0).UTC()
	gz.OS = 255
	tw := tar.NewWriter(gz)
	h := &tar.Header{Name: "batchweaver-doctor/doctor.json", Mode: 0o600, Size: int64(len(data)), ModTime: time.Unix(0, 0).UTC(), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(h); err != nil {
		return err
	}
	if _, err := tw.Write(data); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

package daemon

import (
	"encoding/json"
	"testing"
)

func FuzzDaemonDiscovery(f *testing.F) {
	seed, _ := json.Marshal(Info{
		ProtocolVersion: ProtocolVersion,
		PID:             42,
		Socket:          "/tmp/bwd-seed.sock",
		WorkspaceDigest: "0123456789abcdef",
		Started:         "2026-08-02T00:00:00Z",
		GoOS:            "linux",
	})
	f.Add(seed)
	f.Add([]byte(`{"protocol_version":"batchweaver.daemon/v0","pid":-1}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		var info Info
		if err := json.Unmarshal(data, &info); err != nil {
			return
		}
		encoded, err := json.Marshal(info)
		if err != nil {
			t.Fatalf("marshal accepted discovery record: %v", err)
		}
		var roundTrip Info
		if err := json.Unmarshal(encoded, &roundTrip); err != nil {
			t.Fatalf("decode marshaled discovery record: %v", err)
		}
		if roundTrip.ProtocolVersion != info.ProtocolVersion || roundTrip.Socket != info.Socket || roundTrip.WorkspaceDigest != info.WorkspaceDigest {
			t.Fatal("daemon discovery round trip changed identity fields")
		}
	})
}

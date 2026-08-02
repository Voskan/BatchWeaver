package protocol

import (
	"bytes"
	"encoding/json"
	"testing"
)

func FuzzProtocolInitialize(f *testing.F) {
	f.Add([]byte(`{"processId":7,"clientInfo":{"name":"vscode"},"rootUri":"file:///workspace","capabilities":{}}`))
	f.Add([]byte(`{"workspaceFolders":[],"trace":"off"}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		var params InitializeParams
		if err := json.Unmarshal(data, &params); err != nil {
			return
		}
		encoded, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("marshal accepted initialize params: %v", err)
		}
		var roundTrip InitializeParams
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		if err := decoder.Decode(&roundTrip); err != nil {
			t.Fatalf("decode marshaled initialize params: %v", err)
		}
	})
}

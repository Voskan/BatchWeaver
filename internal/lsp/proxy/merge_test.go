package proxy

import (
	"encoding/json"
	"testing"
)

func TestMergeInitializeResultPreservesGopls(t *testing.T) {
	gopls := json.RawMessage(`{
		"capabilities": {
			"completionProvider": {"triggerCharacters": ["."]},
			"definitionProvider": true,
			"hoverProvider": true,
			"executeCommandProvider": {"commands": ["gopls.tidy"]}
		},
		"serverInfo": {"name": "gopls", "version": "v0.0.0"}
	}`)
	merged := mergeInitializeResult(gopls)
	var out map[string]any
	if err := json.Unmarshal(merged, &out); err != nil {
		t.Fatal(err)
	}
	caps := out["capabilities"].(map[string]any)
	if _, ok := caps["completionProvider"]; !ok {
		t.Error("gopls completionProvider must be preserved")
	}
	if caps["definitionProvider"] != true {
		t.Error("gopls definitionProvider must be preserved")
	}
	if caps["hoverProvider"] != true {
		t.Error("hoverProvider must be advertised")
	}
	cmds := caps["executeCommandProvider"].(map[string]any)["commands"].([]any)
	haveGopls, haveBW := false, false
	for _, c := range cmds {
		switch c.(string) {
		case "gopls.tidy":
			haveGopls = true
		case "batchweaver.scanWorkspace":
			haveBW = true
		}
	}
	if !haveGopls || !haveBW {
		t.Errorf("merged commands must include both gopls and batchweaver: %v", cmds)
	}
}

func TestMergeInitializeResultHandlesMissingCaps(t *testing.T) {
	merged := mergeInitializeResult(json.RawMessage(`{}`))
	var out map[string]any
	if err := json.Unmarshal(merged, &out); err != nil {
		t.Fatal(err)
	}
	caps := out["capabilities"].(map[string]any)
	if caps["hoverProvider"] != true {
		t.Error("hoverProvider must be advertised even when gopls omits capabilities")
	}
}

func TestIsBatchWeaverCommand(t *testing.T) {
	p := New(Options{})
	yes := p.isBatchWeaverCommand(json.RawMessage(`{"command":"batchweaver.scanWorkspace"}`))
	no := p.isBatchWeaverCommand(json.RawMessage(`{"command":"gopls.tidy"}`))
	if !yes || no {
		t.Errorf("classification wrong: yes=%v no=%v", yes, no)
	}
}

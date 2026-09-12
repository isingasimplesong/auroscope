package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Real CLI / deterministic transport contract, not a real-model qualification.
// Run only with the checksum-verified 2.1.269 native distribution, in disposable
// Arch with no external network or real credentials. The server is loopback.
func TestProviderClaudeRealCLIContract(t *testing.T) {
	path := os.Getenv("AUROSCOPE_TEST_REAL_CLAUDE")
	if path == "" {
		t.Skip("set AUROSCOPE_TEST_REAL_CLAUDE to verified Claude Code 2.1.269")
	}
	version, err := exec.Command(path, "--version").Output()
	if err != nil || strings.TrimSpace(string(version)) != "2.1.269 (Claude Code)" {
		t.Fatal("expected Claude Code 2.1.269")
	}
	t.Setenv("PATH", filepath.Dir(path)+":"+os.Getenv("PATH"))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ANTHROPIC_API_KEY", "inert-local-contract-key")
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		data, _ := io.ReadAll(r.Body)
		var request struct {
			Tools     []json.RawMessage `json:"tools"`
			Stream    bool              `json:"stream"`
			Model     string            `json:"model"`
			MaxTokens int               `json:"max_tokens"`
		}
		if json.Unmarshal(data, &request) != nil {
			t.Error("invalid request")
			w.WriteHeader(400)
			return
		}
		if len(request.Tools) != 0 {
			t.Error("real CLI offered model tools")
			w.WriteHeader(400)
			return
		}
		if !bytes.Contains(data, []byte("pkgname=contract-fixture")) {
			t.Error("real CLI did not send bundle")
			w.WriteHeader(400)
			return
		}
		calls.Add(1)
		t.Logf("local request: stream=%v model=%s max_tokens=%d", request.Stream, request.Model, request.MaxTokens)

		if !request.Stream {
			io.WriteString(w, providerEnvelope("anthropic", providerTestReport))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		start := map[string]any{"type": "message_start", "message": map[string]any{"id": "msg_fixture", "type": "message", "role": "assistant", "model": "claude-sonnet-4-6", "content": []any{}, "stop_reason": nil, "stop_sequence": nil, "usage": map[string]int{"input_tokens": 10, "output_tokens": 0}}}
		events := []any{start,
			map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]string{"type": "text", "text": ""}},
			map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]string{"type": "text_delta", "text": providerTestReport}},
			map[string]any{"type": "content_block_stop", "index": 0},
			map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 30}},
			map[string]any{"type": "message_stop"},
		}
		for _, event := range events {
			data, _ := json.Marshal(event)
			kind := event.(map[string]any)["type"]
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, data)
		}
	}))
	defer s.Close()
	t.Setenv("ANTHROPIC_BASE_URL", s.URL)
	bundle := auditBundle{Identity: recipeIdentity{Pkgbase: "contract-fixture"}, Files: []recipeFile{{Path: "PKGBUILD", Text: "pkgname=contract-fixture\n"}}}
	config := runConfig{codexTimeout: 30 * time.Second, stdout: io.Discard, stderr: io.Discard}.withDefaults()
	report, err := auditClaude(bundle, auditConfig{Model: "claude-sonnet-4-6"}, config)
	if err != nil {
		t.Fatalf("production CLI client: %v; local message requests=%d", err, calls.Load())
	}
	if report.Summary != "recipe reviewed" || calls.Load() != 1 {
		t.Fatalf("unexpected result or request count: %d", calls.Load())
	}
	t.Log("Claude Code 2.1.269: production argv accepted, isolated stdin bundle, zero model tools, one simulated request, result envelope validated")
}

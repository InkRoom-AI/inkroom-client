package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicSkillCommandsNeedNoToken(t *testing.T) {
	cases := []struct {
		name, command, tool string
		input               object
	}{
		{"discovery", "tools", "", nil},
		{"models", "models", "list_models", object{}},
		{"capabilities", "capabilities", "help", object{"topic": "quickstart"}},
		{"catalog call", "call", "list_models", object{}},
		{"help call", "call", "help", object{"topic": "billing"}},
		{"public prompt default search", "call", "prompts", object{}},
		{"public prompt search", "call", "prompts", object{"action": "search", "scope": "recommended", "keyword": "coffee"}},
		{"public prompt get", "call", "prompts", object{"action": "get", "id": "public-id"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if _, supplied := r.Header["Authorization"]; supplied {
					t.Error("anonymous request included an Authorization header")
				}
				var payload object
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				if tc.command == "tools" {
					if payload["method"] != "tools/list" {
						t.Errorf("expected tool discovery, got %v", payload)
					}
				} else {
					params, _ := payload["params"].(map[string]any)
					if payload["method"] != "tools/call" || params["name"] != tc.tool || string(marshal(params["arguments"])) != string(marshal(tc.input)) {
						t.Errorf("public request changed: %v", payload)
					}
				}
				_ = json.NewEncoder(w).Encode(object{"jsonrpc": "2.0", "id": payload["id"], "result": object{"structuredContent": object{"scope": "public_catalog", "officialPricing": object{"status": "not_verified"}}}})
			}))
			defer server.Close()
			t.Setenv("INKROOM_MCP_URL", server.URL)
			t.Setenv("INKROOM_MCP_TOKEN", "")
			t.Setenv("INKROOM_STATE_DIR", t.TempDir())
			dir := t.TempDir()
			args := []string{tc.command, "--output", filepath.Join(dir, "output")}
			if tc.command == "call" {
				path := filepath.Join(dir, "arguments.json")
				if err := os.WriteFile(path, marshal(tc.input), 0600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--tool", tc.tool, "--arguments", path)
			}
			if err := run(args); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("expected one read, got %d", calls)
			}
		})
	}
}

func TestPublicSkillDefaultEndpointNeedsNoSettingsOrEnvironment(t *testing.T) {
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Host != "mcp.inkroom.ai" || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected default target or authorization: %s", r.Host)
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"test","result":{"structuredContent":{"scope":"public_catalog"}}}`))
	}))
	defer server.Close()
	transport := server.Client().Transport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	// This test redirects the production hostname to its loopback-only TLS
	// fixture; no production connection or certificate check is weakened.
	transport.TLSClientConfig.InsecureSkipVerify = true
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	original := http.DefaultTransport
	http.DefaultTransport = transport
	defer func() { http.DefaultTransport = original; transport.CloseIdleConnections() }()
	t.Setenv("INKROOM_MCP_URL", "")
	t.Setenv("INKROOM_MCP_TOKEN", "")
	t.Setenv("INKROOM_STATE_DIR", t.TempDir())
	if _, err := toolRequest(object{"jsonrpc": "2.0", "id": "test", "method": "tools/call", "params": object{"name": "list_models", "arguments": object{}}}, filepath.Join(t.TempDir(), "default.response.json"), object{}, true); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("default public request count: %d", calls)
	}
}

func TestPrivateSkillCallsRemainBlockedBeforeNetwork(t *testing.T) {
	for _, tc := range []struct {
		tool  string
		input object
	}{
		{"account", object{}}, {"generation_status", object{}}, {"spending_policy", object{}},
		{"generate_image", object{"action": "preview"}}, {"derive_image", object{"action": "preview"}},
		{"upscale_image", object{"action": "preview"}}, {"reverse_prompt", object{"action": "get"}},
		{"wait_images", object{"requestId": "private"}}, {"prompts", object{"scope": "mine"}},
		{"prompts", object{"scope": "favorites"}}, {"prompts", object{"action": "optimize"}},
		{"prompts", object{"action": 1}}, {"prompts", object{"scope": true}},
		{"prompts", object{"action": nil}}, {"prompts", object{"scope": nil}},
	} {
		t.Run(tc.tool+"/"+string(marshal(tc.input)), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
			defer server.Close()
			t.Setenv("INKROOM_MCP_URL", server.URL)
			t.Setenv("INKROOM_MCP_TOKEN", "")
			t.Setenv("INKROOM_STATE_DIR", t.TempDir())
			dir := t.TempDir()
			arguments := filepath.Join(dir, "arguments.json")
			if err := os.WriteFile(arguments, marshal(tc.input), 0600); err != nil {
				t.Fatal(err)
			}
			err := run([]string{"call", "--tool", tc.tool, "--arguments", arguments, "--output", filepath.Join(dir, "output")})
			if err == nil || !strings.Contains(err.Error(), "account authorization required") || calls != 0 {
				t.Fatalf("private call bypassed authorization: err=%v calls=%d", err, calls)
			}
		})
	}
}

func TestPublicSkillRejectsSuppliedOrSavedInvalidTokenWithoutAnonymousRetry(t *testing.T) {
	for _, saved := range []bool{false, true} {
		t.Run(map[bool]string{false: "environment", true: "saved"}[saved], func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get("Authorization") != "Bearer synthetic-invalid" {
					t.Error("supplied authorization was dropped")
				}
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"message":"invalid token"}}`))
			}))
			defer server.Close()
			t.Setenv("INKROOM_MCP_URL", server.URL)
			t.Setenv("INKROOM_MCP_TOKEN", "synthetic-invalid")
			t.Setenv("INKROOM_STATE_DIR", t.TempDir())
			if saved {
				t.Setenv("INKROOM_MCP_TOKEN", "")
				if err := atomic(tokenPath(server.URL), []byte("synthetic-invalid")); err != nil {
					t.Fatal(err)
				}
			}
			err := run([]string{"models", "--output", t.TempDir()})
			if err == nil || calls != 1 {
				t.Fatalf("invalid token silently retried: err=%v calls=%d", err, calls)
			}
		})
	}
}

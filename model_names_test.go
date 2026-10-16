package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Exercise the real CLI request path without generation or upstream access.
func TestSkillPreservesPublicAndLegacyModelArguments(t *testing.T) {
	cases := []struct {
		name  string
		tool  string
		input object
	}{
		{"public model", "generate_image", object{"action": "preview", "model": "gemini-2.5-flash-image", "prompt": "original prompt"}},
		{"original model", "generate_image", object{"action": "preview", "model": "nano-upstream-image-v1", "prompt": "original prompt"}},
		{"historical alias", "derive_image", object{"action": "preview", "model": "nano-previous-public-name", "prompt": "original prompt"}},
		{"mixed model comparison", "batch_generate", object{"action": "preview", "items": []any{
			object{"model": "gemini-2.5-flash-image", "prompt": "same prompt"},
			object{"model": "nano-upstream-image-v1", "prompt": "same prompt"},
			object{"model": "nano-previous-public-name", "prompt": "same prompt"},
		}}},
		{"server default", "generate_image", object{"action": "preview", "prompt": "original prompt"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got object
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Error(err)
				}
				_ = json.NewEncoder(w).Encode(object{"jsonrpc": "2.0", "id": got["id"], "result": object{
					"structuredContent": object{"model": "gemini-2.5-flash-image", "maxPoints": 5},
				}})
			}))
			defer server.Close()
			t.Setenv("INKROOM_MCP_URL", server.URL)
			t.Setenv("INKROOM_MCP_TOKEN", "synthetic-model-test")
			dir := t.TempDir()
			arguments := filepath.Join(dir, "arguments.json")
			if err := os.WriteFile(arguments, marshal(tc.input), 0600); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(dir, "output")
			if err := run([]string{"call", "--tool", tc.tool, "--arguments", arguments, "--output", output}); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || got["method"] != "tools/call" {
				t.Fatalf("unexpected request path: calls=%d, request=%v", calls, got)
			}
			params, _ := got["params"].(map[string]any)
			want := object{}
			_ = json.Unmarshal(marshal(tc.input), &want)
			want["deliveryMode"] = "links"
			if params["name"] != tc.tool || !reflect.DeepEqual(params["arguments"], want) {
				t.Fatalf("model or other caller arguments changed: got=%v, want=%v", params, want)
			}
			records, _ := filepath.Glob(filepath.Join(output, "*.response.json"))
			if len(records) != 1 {
				t.Fatalf("expected one server response, got %v", records)
			}
			raw, err := os.ReadFile(records[0])
			if err != nil {
				t.Fatal(err)
			}
			response, err := parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			result := response["result"].(map[string]any)["structuredContent"].(map[string]any)
			if result["model"] != "gemini-2.5-flash-image" || result["maxPoints"] != float64(5) {
				t.Fatalf("server public name or pricing was not preserved: %v", result)
			}
		})
	}
}

func TestSkillKeepsDynamicPublicModelCatalog(t *testing.T) {
	catalog := object{"models": []any{object{"model": "new-configured-public-model", "pointCost": 7}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload object
		_ = json.NewDecoder(r.Body).Decode(&payload)
		params, _ := payload["params"].(map[string]any)
		if payload["method"] != "tools/call" || params["name"] != "list_models" {
			t.Errorf("unexpected catalog request: %v", payload)
		}
		_ = json.NewEncoder(w).Encode(object{"jsonrpc": "2.0", "id": payload["id"], "result": object{"structuredContent": catalog}})
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "synthetic-model-test")
	dir := t.TempDir()
	arguments := filepath.Join(dir, "arguments.json")
	if err := os.WriteFile(arguments, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "output")
	if err := run([]string{"call", "--tool", "list_models", "--arguments", arguments, "--output", output}); err != nil {
		t.Fatal(err)
	}
	records, _ := filepath.Glob(filepath.Join(output, "*.response.json"))
	if len(records) != 1 {
		t.Fatal("catalog response missing")
	}
	raw, _ := os.ReadFile(records[0])
	response, err := parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	actual := response["result"].(map[string]any)["structuredContent"]
	var want any
	_ = json.Unmarshal(marshal(catalog), &want)
	if !reflect.DeepEqual(compact(actual), want) {
		t.Fatalf("client altered dynamic model catalog: %v", actual)
	}
}

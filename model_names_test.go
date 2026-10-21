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
func TestSkillExecutionReceiptsAreSentWithoutDefaultFields(t *testing.T) {
	for _, tool := range []string{"generate_image", "derive_image", "upscale_image", "batch_generate"} {
		for _, action := range []string{"auto", "run"} {
			t.Run(tool+"/"+action, func(t *testing.T) {
				input := object{"action": action, "requestId": "original-preview"}
				if tool == "batch_generate" {
					delete(input, "requestId")
					input["batchId"] = "original-batch"
				}
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					var payload object
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					params, _ := payload["params"].(map[string]any)
					if payload["method"] != "tools/call" || params["name"] != tool || !reflect.DeepEqual(params["arguments"], input) {
						t.Errorf("receipt arguments changed: %v", payload)
					}
					_ = json.NewEncoder(w).Encode(object{"jsonrpc": "2.0", "id": payload["id"], "result": object{"structuredContent": object{"status": "running"}}})
				}))
				defer server.Close()
				t.Setenv("INKROOM_MCP_URL", server.URL)
				t.Setenv("INKROOM_MCP_TOKEN", "synthetic-receipt-test")
				dir := t.TempDir()
				arguments := filepath.Join(dir, "arguments.json")
				if err := os.WriteFile(arguments, marshal(input), 0600); err != nil {
					t.Fatal(err)
				}
				if err := run([]string{"call", "--tool", tool, "--arguments", arguments, "--output", filepath.Join(dir, "output")}); err != nil {
					t.Fatal(err)
				}
				if calls != 1 {
					t.Fatalf("execution was sent %d times", calls)
				}
			})
		}
	}
}

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
		{"native preview", "generate_image", object{"action": "preview", "prompt": "original prompt", "deliveryMode": "native"}},
		{"execute single preview", "generate_image", object{"action": "auto", "requestId": "original-preview"}},
		{"execute edit preview", "derive_image", object{"action": "run", "requestId": "original-preview"}},
		{"execute processing preview", "upscale_image", object{"action": "auto", "requestId": "original-preview"}},
		{"execute batch preview", "batch_generate", object{"action": "run", "batchId": "original-batch"}},
		{"receipt wait", "wait_images", object{"batchId": "original-batch", "cursor": "original-cursor"}},
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
			if (tc.input["requestId"] == nil && tc.input["batchId"] == nil || tc.tool == "wait_images") && tc.input["deliveryMode"] == nil {
				want["deliveryMode"] = "links"
			}
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

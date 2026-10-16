package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDeliveryFailureClassificationAndIdentity(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{{mediaTransportError(context.DeadlineExceeded), "download_timeout"}, {&mediaFailure{Code: "download_http", HTTPStatus: 404}, "download_http"}, {&mediaFailure{Code: "local_write_failed"}, "local_write_failed"}} {
		entry := mediaFailureEntry(test.err, "original_link", object{"taskId": "original", "index": float64(0), "resultId": "img_1"})
		if entry["errorCode"] != test.want || entry["taskId"] != "original" {
			t.Fatal(entry)
		}
	}
}
func TestSkillReportsOnlyDefinitiveOriginalFailuresAndRetainsEvidence(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var payload object
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		params := payload["params"].(map[string]any)
		if params["name"] != "report_image_delivery" {
			t.Error("unexpected tool", params)
		}
		args := params["arguments"].(map[string]any)
		if args["taskId"] != "original" || args["imageIndex"] != float64(0) || args["url"] != nil {
			t.Error("unsafe review", args)
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"structuredContent":{"status":"refund_pending"}}}`))
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "synthetic")
	entries := []object{
		mediaFailureEntry(&mediaFailure{Code: "download_http", HTTPStatus: 404}, "original_link", object{"taskId": "original", "index": 0, "resultId": "img_1"}),
		mediaFailureEntry(&mediaFailure{Code: "download_timeout"}, "original_link", object{}),
		mediaFailureEntry(&mediaFailure{Code: "local_write_failed"}, "original_link", object{}),
		mediaFailureEntry(&mediaFailure{Code: "download_http", HTTPStatus: 502}, "original_link", object{}),
	}
	record := filepath.Join(t.TempDir(), "original.response.json")
	reviewFailedMedia(entries, record, filepath.Dir(record), object{})
	if calls != 1 || entries[0]["deliveryReview"] == nil {
		t.Fatal(calls, entries)
	}
	for _, suffix := range []string{".request.json", ".response.json"} {
		matches, err := filepath.Glob(filepath.Join(filepath.Dir(record), "original.response.delivery-0-*"+suffix))
		if err != nil || len(matches) != 1 {
			t.Fatal(matches, err)
		}
	}
}
func TestRecoveryExitsNonzeroForInvalidMediaWithoutGenerating(t *testing.T) {
	dir := t.TempDir()
	response := filepath.Join(dir, "response.json")
	if err := os.WriteFile(response, []byte(`{"result":{"content":[{"type":"image","data":"badbase64"}]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"recover", "--response", response, "--output", dir}); err == nil {
		t.Fatal("media failure exited successfully")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "response.manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest []object
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest[0]["errorCode"] != "decode_failed" {
		t.Fatal(manifest)
	}
}

func TestDuplicateStructuredURLKeepsResourceIdentity(t *testing.T) {
	input := object{"result": object{"structuredContent": object{"url": "https://static.example.test/image", "width": 2}, "content": []any{object{"type": "resource_link", "uri": "https://static.example.test/image", "_meta": object{"inkroom/image": object{"taskId": "task", "index": 0, "resultId": "result"}}}}}}
	identities := mediaIdentities(input)
	// object is the JSON map alias used by parsed responses.
	raw, _ := json.Marshal(input)
	var decoded object
	_ = json.Unmarshal(raw, &decoded)
	identities = mediaIdentities(decoded)
	if identities["https://static.example.test/image"]["taskId"] != "task" {
		t.Fatal(identities)
	}
}

func TestCompletedImageWithoutLinkIsNotSilentSuccess(t *testing.T) {
	value := object{"result": object{"structuredContent": object{"resultManifest": object{"inputs": []any{object{"index": 0, "taskId": "original"}}, "results": []any{object{"inputIndex": 0, "imageIndex": 0, "resultId": "img_missing", "status": "ready", "deliveryStatus": "unavailable"}, object{"inputIndex": 0, "imageIndex": 1, "resultId": "img_pending", "status": "pending"}}}}}}
	missing := unavailableOriginals(value)
	if len(missing) != 1 || missing[0]["taskId"] != "original" || missing[0]["saved"] != false {
		t.Fatal(missing)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"structuredContent":{"status":"unconfirmed"}}}`))
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "synthetic")
	dir := t.TempDir()
	response := filepath.Join(dir, "response.json")
	raw, _ := json.Marshal(value)
	_ = os.WriteFile(response, raw, 0600)
	if err := run([]string{"recover", "--response", response, "--output", dir}); err == nil {
		t.Fatal("unavailable originals reported success")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "response.delivery-*.request.json"))
	if len(matches) != 1 {
		t.Fatal("missing verification request", matches)
	}
}

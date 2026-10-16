package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOriginalDownloadResumesInterruptedBody(t *testing.T) {
	data := bytes.Repeat([]byte("verified-original"), 32)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "" {
			t.Error("credential forwarded")
		}
		w.Header().Set("ETag", `"immutable"`)
		if calls == 1 {
			w.Header().Set("Content-Length", fmt.Sprint(len(data)))
			w.Write(data[:23])
			return
		}
		if r.Header.Get("Range") != "bytes=23-" || r.Header.Get("If-Range") != `"immutable"` {
			t.Error("missing safe resume", r.Header)
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 23-%d/%d", len(data)-1, len(data)))
		w.Header().Set("Content-Length", fmt.Sprint(len(data)-23))
		w.WriteHeader(206)
		w.Write(data[23:])
	}))
	defer server.Close()
	got, err := downloadOriginalWithClient(context.Background(), server.Client(), server.URL, t.TempDir(), object{"resultId": "same-original"})
	if err != nil || !bytes.Equal(got, data) || calls != 2 {
		t.Fatal("resume failed", err, calls)
	}
}
func TestOriginalDownloadRejectsEmptyAndMalformedRange(t *testing.T) {
	for _, status := range []int{200, 206} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if status == 206 {
					w.Header().Set("Content-Range", "bytes 5-9/10")
				}
				w.WriteHeader(status)
			}))
			defer server.Close()
			_, err := downloadOriginalWithClient(context.Background(), server.Client(), server.URL, t.TempDir(), object{"resultId": "empty"})
			var failure *mediaFailure
			if !errors.As(err, &failure) || (status == 200 && failure.Code != "download_empty") || (status == 206 && failure.Code != "download_range_invalid") {
				t.Fatal(err)
			}
		})
	}
}
func TestSuccessfulManifestRetainsEveryResultEvenWithSameHash(t *testing.T) {
	dir := t.TempDir()
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(imageBytes.Bytes())
	digest := hex.EncodeToString(sum[:])
	path := filepath.Join(dir, digest+".png")
	if err := atomic(path, imageBytes.Bytes()); err != nil {
		t.Fatal(err)
	}
	blocks := []any{}
	for index := 0; index < 2; index++ {
		link := "https://static.example.test/original?signature=" + fmt.Sprint(index)
		identity := object{"taskId": "task", "index": index, "inputIndex": 4, "resultId": fmt.Sprint("result-", index)}
		if err := checkpointOriginal(dir, link, identity, path, digest); err != nil {
			t.Fatal(err)
		}
		blocks = append(blocks, object{"type": "resource_link", "uri": link, "_meta": object{"inkroom/image": identity}})
	}
	raw, _ := json.Marshal(object{"result": object{"content": blocks}})
	var value object
	_ = json.Unmarshal(raw, &value)
	entries := saveMedia(value, dir)
	if len(entries) != 2 {
		t.Fatalf("collapsed different results: %v", entries)
	}
	for index, entry := range entries {
		if entry["saved"] != true || entry["resultId"] != fmt.Sprint("result-", index) || entry["inputIndex"] != float64(4) || entry["imageIndex"] != float64(index) || entry["mediaKind"] != "original" {
			t.Fatal(entry)
		}
	}
	if entries[0]["path"] != entries[1]["path"] {
		t.Fatal("expected physical content deduplication")
	}
}
func TestResumeFollowsOnlyReadOnlyOriginalContinuation(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var payload object
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		params := payload["params"].(map[string]any)
		if params["name"] != "wait_images" {
			t.Error("paid operation during resume")
		}
		args := params["arguments"].(map[string]any)
		if args["batchId"] != "original" || args["cursor"] != "cursor-exact" || args["deliveryMode"] != "links" {
			t.Error(args)
		}
		json.NewEncoder(w).Encode(object{"result": object{"structuredContent": object{"complete": true}}})
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "synthetic")
	dir := t.TempDir()
	response := filepath.Join(dir, "original.response.json")
	value := object{"result": object{"structuredContent": object{"nextAction": object{"tool": "wait_images", "arguments": object{"batchId": "original", "cursor": "cursor-exact", "deliveryMode": "links"}}}}}
	if err := atomic(response, marshal(value)); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"resume", "--response", response, "--output", dir}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	checkpoint, err := os.ReadFile(strings.TrimSuffix(response, ".json") + ".resume.json")
	if err != nil || !bytes.Contains(checkpoint, []byte("latestResponse")) {
		t.Fatal("checkpoint missing", err)
	}
	records, _ := filepath.Glob(filepath.Join(dir, "*.response.json"))
	if len(records) != 2 {
		t.Fatal("continuation response not saved", records)
	}
	_, ok := continuation(object{"result": object{"structuredContent": object{"nextAction": object{"tool": "batch_generate", "arguments": object{"action": "run"}}}}})
	if ok {
		t.Fatal("paid continuation accepted")
	}
}

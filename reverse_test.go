package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func reversePNG(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestReverseLocalPreviewUsesSharedAuthorizationAndPreservesSource(t *testing.T) {
	raw := reversePNG(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "input.png")
	if err := os.WriteFile(source, raw, 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer synthetic-shared-generation-token" {
			t.Error("reverse did not reuse generation token")
		}
		if r.Header.Get("X-InkRoom-Client") != "skill" {
			t.Error("reverse request did not preserve Skill billing display attribution")
		}
		var payload object
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		params, _ := payload["params"].(map[string]any)
		args, _ := params["arguments"].(map[string]any)
		want := object{"action": "preview", "image": "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)}
		if params["name"] != "reverse_prompt" || !reflect.DeepEqual(args, want) {
			t.Errorf("reverse preview changed arguments: %v", payload)
		}
		// A private/unreachable input source is metadata, never output media.
		_ = json.NewEncoder(w).Encode(object{"result": object{"structuredContent": object{"requestId": "reverse-1", "status": "preview", "preview": object{"model": "admin-vision", "estimatedChargePoints": 7, "source": object{"url": "https://127.0.0.1/input.png", "width": 2, "height": 3}}}}})
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "")
	t.Setenv("INKROOM_STATE_DIR", filepath.Join(dir, "account-state"))
	if err := atomic(tokenPath(server.URL), []byte("synthetic-shared-generation-token")); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "result")
	if err := run([]string{"reverse", "--image", source, "--output", out}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("preview triggered %d remote calls", calls)
	}
	after, _ := os.ReadFile(source)
	if !bytes.Equal(after, raw) {
		t.Fatal("source image was modified")
	}
	records, _ := filepath.Glob(filepath.Join(out, "*.response.json"))
	if len(records) != 1 {
		t.Fatal("missing saved preview receipt", records)
	}
	resultRaw, _ := os.ReadFile(records[0])
	if !bytes.Contains(resultRaw, []byte(`"estimatedChargePoints":7`)) {
		t.Fatal("administrator-selected quote was replaced")
	}
	manifest, _ := os.ReadFile(strings.TrimSuffix(records[0], ".json") + ".manifest.json")
	if string(manifest) != "[]" {
		t.Fatalf("input image was treated as generated media: %s", manifest)
	}
}

func TestReverseExecutionAndRecoveryOnlySendOriginalReceipt(t *testing.T) {
	for _, action := range []string{"auto", "run", "get"} {
		t.Run(action, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get("Authorization") != "Bearer synthetic-env-token" {
					t.Error("environment authorization was not reused")
				}
				if r.Header.Get("X-InkRoom-Client") != "skill" {
					t.Error("receipt recovery lost Skill display attribution")
				}
				var payload object
				_ = json.NewDecoder(r.Body).Decode(&payload)
				params := payload["params"].(map[string]any)
				if params["name"] != "reverse_prompt" || !reflect.DeepEqual(params["arguments"], object{"action": action, "requestId": "frozen-quote"}) {
					t.Errorf("execution/recovery changed frozen arguments: %v", payload)
				}
				_ = json.NewEncoder(w).Encode(object{"result": object{"structuredContent": object{"status": "succeeded", "requestId": "frozen-quote", "result": object{"positivePromptEn": "Exact returned prompt."}}}})
			}))
			defer server.Close()
			t.Setenv("INKROOM_MCP_URL", server.URL)
			t.Setenv("INKROOM_MCP_TOKEN", "synthetic-env-token")
			if err := run([]string{"reverse", "--action", action, "--request-id", "frozen-quote", "--output", t.TempDir()}); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("execution/recovery triggered %d calls", calls)
			}
		})
	}
}

func TestReverseRejectsAmbiguousOrInvalidInputsBeforeNetwork(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "synthetic")
	for _, args := range [][]string{
		{}, {"--action", "unknown"}, {"--action", "get"}, {"--action", "auto", "--request-id", ""},
		{"--image", "https://example.test/a.png", "--request-id", "existing"},
		{"--action", "get", "--request-id", "existing", "--image", "https://example.test/a.png"},
		{"--image", "https://example.test/a.png", "--image", "https://example.test/b.png"},
		{"--image", "https://example.test/a.png", "second.png"}, {"--image", "data:image/png;base64,bad"},
		{"--image", "https://user:pass@example.test/a.png"}, {"--image", "https://example.test/a.png#secret"},
	} {
		if err := run(append([]string{"reverse"}, args...)); err == nil {
			t.Errorf("invalid reverse request accepted: %v", args)
		}
	}
	if calls != 0 {
		t.Fatalf("invalid inputs made %d requests", calls)
	}
}

func TestReverseLocalImageValidation(t *testing.T) {
	raw := reversePNG(t)
	var jpegImage bytes.Buffer
	if err := jpeg.Encode(&jpegImage, image.NewRGBA(image.Rect(0, 0, 2, 3)), nil); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		raw  []byte
		mime string
	}{{"png", raw, "image/png"}, {"jpeg", jpegImage.Bytes(), "image/jpeg"}} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reverseImageDataURL(tc.raw)
			if err != nil || got != "data:"+tc.mime+";base64,"+base64.StdEncoding.EncodeToString(tc.raw) {
				t.Fatalf("image bytes or MIME changed: %v", err)
			}
			if again, err := reverseImageSource(got); err != nil || got != again {
				t.Fatalf("data URL changed: %v", err)
			}
		})
	}
	oversized := append([]byte(nil), raw...)
	binary.BigEndian.PutUint32(oversized[16:20], reverseImageMaxPixels+1)
	binary.BigEndian.PutUint32(oversized[20:24], 1)
	binary.BigEndian.PutUint32(oversized[29:33], crc32.ChecksumIEEE(oversized[12:29]))
	for name, content := range map[string][]byte{"empty": nil, "invalid": []byte("text"), "truncated": raw[:len(raw)/2], "pixel-bomb": oversized, "byte-limit": make([]byte, reverseImageMaxBytes+1)} {
		t.Run(name, func(t *testing.T) {
			if _, err := reverseImageDataURL(content); err == nil {
				t.Fatal("invalid or oversized image accepted")
			}
		})
	}
	dir := t.TempDir()
	if _, err := reverseImageSource(dir); err == nil {
		t.Fatal("directory accepted as image")
	}
	file := filepath.Join(dir, "too-large.png")
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Truncate(reverseImageMaxBytes + 1)
	_ = f.Close()
	if _, err := reverseImageSource(file); err == nil {
		t.Fatal("oversized local file accepted")
	}
	if _, err := reverseImageSource("data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("mismatched data URL type accepted")
	}
}

func TestReverseFailureRetainsReceiptWithoutAnotherCall(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"result":{"isError":true,"structuredContent":{"requestId":"original","status":"confirmation_required"}}}`))
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "synthetic")
	out := t.TempDir()
	if err := run([]string{"reverse", "--action", "auto", "--request-id", "original", "--output", out}); err == nil {
		t.Fatal("tool error was not reported")
	}
	if calls != 1 {
		t.Fatal("failed reverse retried or generated an image", calls)
	}
	for _, pattern := range []string{"*.request.json", "*.response.json"} {
		files, _ := filepath.Glob(filepath.Join(out, pattern))
		if len(files) != 1 {
			t.Fatal("missing request/response evidence", files)
		}
	}
}

func TestGenericCallForReverseDoesNotAddImageDeliveryOrDownloadSource(t *testing.T) {
	input := object{"action": "preview", "image": "https://images.example.test/input.png"}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var payload object
		_ = json.NewDecoder(r.Body).Decode(&payload)
		params := payload["params"].(map[string]any)
		if params["name"] != "reverse_prompt" || !reflect.DeepEqual(params["arguments"], input) {
			t.Errorf("generic reverse tool call was changed: %v", payload)
		}
		_, _ = w.Write([]byte(`{"result":{"structuredContent":{"preview":{"source":{"url":"https://127.0.0.1/input.png","width":2,"height":3}}}}}`))
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "synthetic")
	dir := t.TempDir()
	arguments := filepath.Join(dir, "arguments.json")
	if err := os.WriteFile(arguments, marshal(input), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"call", "--tool", "reverse_prompt", "--arguments", arguments, "--output", filepath.Join(dir, "output")}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("reverse triggered extra remote activity", calls)
	}
}

func TestReverseSkillSourcesStayInSync(t *testing.T) {
	for _, path := range []string{"PROMPT_REVERSE_SKILL.md"} {
		data, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(data, promptReverseSkill) {
			t.Errorf("reverse instructions differ from the embedded source: %s: %v", path, err)
		}
	}
}

func TestReverseInstallerPreservesGenerationSkillAndSharesEndpoint(t *testing.T) {
	root := t.TempDir()
	common := []string{"--api-origin", "https://api.example.test", "--mcp-url", "https://mcp.example.test", "--skills-dir", root}
	if err := install(common); err != nil {
		t.Fatal(err)
	}
	old, _ := os.ReadFile(filepath.Join(root, "inkroom", "SKILL.md"))
	if err := install(append(common, "--skill", "inkroom-prompt-reverse")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "inkroom-prompt-reverse", "SKILL.md"))
	if !bytes.Equal(got, promptReverseSkill) || bytes.Equal(got, skill) {
		t.Fatal("installer selected the wrong embedded Skill")
	}
	firstSettings, _ := os.ReadFile(filepath.Join(root, "inkroom", "settings.json"))
	secondSettings, _ := os.ReadFile(filepath.Join(root, "inkroom-prompt-reverse", "settings.json"))
	if !bytes.Equal(firstSettings, secondSettings) {
		t.Fatal("reverse does not share generation endpoints")
	}
	after, _ := os.ReadFile(filepath.Join(root, "inkroom", "SKILL.md"))
	if !bytes.Equal(old, after) {
		t.Fatal("reverse installation modified the generation Skill")
	}
	if err := install(append(common, "--skill", "inkroom-prompt-reverse")); err == nil {
		t.Fatal("existing reverse Skill overwritten")
	}
	if err := install(append(common, "--skill", "../other")); err == nil {
		t.Fatal("arbitrary Skill install path accepted")
	}
}

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeAndRecoverOriginal(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	img := object{"type": "image", "data": base64.StdEncoding.EncodeToString(b.Bytes())}
	dir := t.TempDir()
	result := saveMedia(object{"content": []any{img, img}}, dir)
	if len(result) != 1 || result[0]["width"] != 2 || result[0]["height"] != 3 || result[0]["decoded"] != true {
		t.Fatalf("%v", result)
	}
	again := saveMedia(img, dir)
	if again[0]["sha256"] != result[0]["sha256"] {
		t.Fatal("recovery changed image")
	}
}
func TestGuestCommandsRejectedBeforeNetwork(t *testing.T) {
	for _, command := range []string{"guest-access", "guest-generate"} {
		if err := run([]string{command}); err == nil {
			t.Fatal("guest command accepted")
		}
	}
}

func TestInstallerDiscoversEndpointsAndPreservesExisting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(object{"code": 0, "data": object{"endpoints": object{"apiOrigin": "https://api.example.org", "mcpUrl": "https://mcp.example.org"}}})
	}))
	defer server.Close()
	root := t.TempDir()
	args := []string{"--origin", server.URL, "--skills-dir", root}
	if err := install(args); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "inkroom", "settings.json")
	raw, _ := os.ReadFile(path)
	if !bytes.Contains(raw, []byte("https://api.example.org")) {
		t.Fatal("endpoints missing")
	}
	if err := install(args); err == nil {
		t.Fatal("existing installation overwritten")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(raw, after) {
		t.Fatal("configuration altered")
	}
}
func TestSSEAndErrorRetention(t *testing.T) {
	v, err := parse([]byte("event: message\r\ndata: {\"result\":\r\ndata: {\"nextAction\":{\"arguments\":{\"cursor\":\"c1\"}}}}\r\n\r\n"))
	if err != nil || v["result"] == nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"error":{"message":"denied"}}`))
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "synthetic")
	dir := t.TempDir()
	if err := run([]string{"tools", "--output", dir}); err == nil || !strings.Contains(err.Error(), "没有访问权限") || !strings.Contains(err.Error(), ".response.json") {
		t.Fatalf("403 did not explain authorization failure: %v", err)
	}
	records, _ := filepath.Glob(filepath.Join(dir, "*.response.json"))
	if len(records) != 1 {
		t.Fatal("error response not saved")
	}
}
func TestAuthorizeRejectsInvalidTokenWithoutSaving(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("MCP Token 无效或已撤销"))
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_STATE_DIR", t.TempDir())
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.WriteString("synthetic-invalid-token\n")
	_ = writer.Close()
	previous := os.Stdin
	os.Stdin = reader
	defer func() { os.Stdin = previous; _ = reader.Close() }()
	if err := run([]string{"authorize"}); err == nil || !strings.Contains(err.Error(), "Token 无效") {
		t.Fatalf("invalid token was not rejected clearly: %v", err)
	}
	if _, err := os.Stat(tokenPath(server.URL)); !os.IsNotExist(err) {
		t.Fatalf("invalid token was saved: %v", err)
	}
}
func TestAuthorizeVerifiesAndStoresTokenPrivately(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-valid-token" {
			t.Error("authorization header missing")
		}
		_ = json.NewEncoder(w).Encode(object{"jsonrpc": "2.0", "id": "test", "result": object{"tools": []any{}}})
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_STATE_DIR", t.TempDir())
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = writer.WriteString("synthetic-valid-token\n")
	_ = writer.Close()
	previous := os.Stdin
	os.Stdin = reader
	defer func() { os.Stdin = previous; _ = reader.Close() }()
	if err := run([]string{"authorize"}); err != nil {
		t.Fatal(err)
	}
	path := tokenPath(server.URL)
	stored, err := os.ReadFile(path)
	if err != nil || string(stored) != "synthetic-valid-token" {
		t.Fatal("verified token was not stored")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("token file permissions are not private: %v", err)
	}
}
func TestPrivateMediaRejected(t *testing.T) {
	if _, err := request("https://127.0.0.1/private", nil, nil, true); err == nil {
		t.Fatal("private URL accepted")
	}
}

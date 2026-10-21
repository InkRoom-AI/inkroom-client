package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	syncatomic "sync/atomic"
	"testing"
)

func TestAccountRequestRetainsReceivedEmptyAndInvalidResponses(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "empty success", status: http.StatusOK},
		{name: "non JSON success", status: http.StatusOK, body: "<html>upstream unavailable</html>"},
		{name: "empty HTTP failure", status: http.StatusBadGateway},
		{name: "non JSON HTTP failure", status: http.StatusBadGateway, body: "upstream unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls syncatomic.Int32
			payload := object{"jsonrpc": "2.0", "id": "original-rpc-id", "method": "tools/call", "params": object{"name": "reverse_prompt", "arguments": object{"action": "auto", "requestId": "original-reverse-id"}}}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var received object
				if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
					t.Error(err)
				}
				if !bytes.Equal(marshal(received), marshal(payload)) {
					t.Errorf("original submitted request changed: %v", received)
				}
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			t.Setenv("INKROOM_MCP_URL", server.URL)
			t.Setenv("INKROOM_MCP_TOKEN", "synthetic-test-token")
			record := filepath.Join(t.TempDir(), "original.response.json")
			_, err := accountRequest(payload, record, object{})
			if err == nil || !strings.Contains(err.Error(), record) || !strings.Contains(err.Error(), strings.TrimSuffix(record, ".response.json")+".request.json") {
				t.Fatalf("missing actual receipt paths: %v", err)
			}
			if test.status == http.StatusOK && (!strings.Contains(err.Error(), "执行结果未知") || !strings.Contains(err.Error(), "原 requestId：original-reverse-id") || !strings.Contains(err.Error(), "勿重新提交")) {
				t.Fatalf("invalid result omitted original-request recovery instructions: %v", err)
			}
			if calls.Load() != 1 {
				t.Fatalf("possibly submitted request retried %d times", calls.Load())
			}
			actual, readErr := os.ReadFile(record)
			if readErr != nil || !bytes.Equal(actual, []byte(test.body)) {
				t.Fatalf("response evidence changed or missing: %q, %v", actual, readErr)
			}
			info, statErr := os.Stat(record)
			if statErr != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("response evidence is not private: %v", statErr)
			}
			requestRaw, readErr := os.ReadFile(strings.TrimSuffix(record, ".response.json") + ".request.json")
			if readErr != nil || !bytes.Equal(requestRaw, marshal(payload)) {
				t.Fatalf("original request evidence changed or missing: %v", readErr)
			}
		})
	}
}

func TestAccountRequestPreservesPartialBodyWithoutRetry(t *testing.T) {
	var calls syncatomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer connection.Close()
		_, _ = fmt.Fprint(connection, "HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\npartial")
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "synthetic-test-token")
	record := filepath.Join(t.TempDir(), "original.response.json")
	_, err := accountRequest(object{"method": "tools/call"}, record, object{})
	if err == nil || !strings.Contains(err.Error(), "response interrupted") || !strings.Contains(err.Error(), record) {
		t.Fatalf("partial response failure lost original evidence: %v", err)
	}
	actual, readErr := os.ReadFile(record)
	if readErr != nil || string(actual) != "partial" || calls.Load() != 1 {
		t.Fatalf("partial response changed or request retried: %q, %v, %d", actual, readErr, calls.Load())
	}
}

func TestAccountRequestTransportFailureDoesNotInventResponseReceipt(t *testing.T) {
	var calls syncatomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = connection.Close()
	}))
	defer server.Close()
	t.Setenv("INKROOM_MCP_URL", server.URL)
	t.Setenv("INKROOM_MCP_TOKEN", "synthetic-test-token")
	record := filepath.Join(t.TempDir(), "original.response.json")
	_, err := accountRequest(object{"method": "tools/call"}, record, object{})
	if err == nil || !strings.Contains(err.Error(), "connection failed") || !strings.Contains(err.Error(), "do not automatically resubmit") || strings.Contains(err.Error(), "响应记录") || strings.Contains(err.Error(), "retained") {
		t.Fatalf("transport failure invented response evidence or permitted resubmission: %v", err)
	}
	if _, statErr := os.Stat(record); !os.IsNotExist(statErr) {
		t.Fatalf("invented response file: %v", statErr)
	}
	requestRecord := strings.TrimSuffix(record, ".response.json") + ".request.json"
	if _, statErr := os.Stat(requestRecord); statErr != nil || !strings.Contains(err.Error(), requestRecord) || calls.Load() != 1 {
		t.Fatalf("missing original request or request retried: %v, %d", statErr, calls.Load())
	}
}

func TestParseDoesNotClaimResponseWasSaved(t *testing.T) {
	for _, raw := range [][]byte{nil, {}, []byte("not JSON")} {
		_, err := parse(raw)
		if err == nil || err.Error() != "no JSON result" {
			t.Fatalf("parser claimed receipt retention without saving: %v", err)
		}
	}
}

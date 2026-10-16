package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

type mediaFailure struct {
	Code       string
	HTTPStatus int
}

func (e *mediaFailure) Error() string {
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("%s HTTP %d", e.Code, e.HTTPStatus)
	}
	return e.Code
}
func mediaTransportError(err error) error {
	var timeout net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
		return &mediaFailure{Code: "download_timeout"}
	}
	return &mediaFailure{Code: "download_transport"}
}
func mediaFailureEntry(err error, source string, identity object) object {
	item := object{"saved": false, "source": source, "error": "image download or decode failed", "errorCode": "decode_failed"}
	var failure *mediaFailure
	if errors.As(err, &failure) {
		item["errorCode"] = failure.Code
		if failure.HTTPStatus != 0 {
			item["httpStatus"] = failure.HTTPStatus
		}
	}
	for k, v := range identity {
		item[k] = v
	}
	return item
}
func mediaIdentity(n map[string]any) object {
	meta, _ := n["_meta"].(map[string]any)
	identity, _ := meta["inkroom/image"].(map[string]any)
	out := object{}
	for _, key := range []string{"taskId", "index", "resultId", "inputIndex"} {
		if v, ok := identity[key]; ok {
			out[key] = v
			if key == "index" {
				out["imageIndex"] = v
			}
		}
	}
	return out
}

// Report only original-link HTTP failures with server-issued image identities.
// Save exact review request/response before parsing. Never sends a URL, price or
// client error as refund proof; only the server can confirm compensation.
func reviewFailedMedia(entries []object, record, output string, config object) {
	mcp := env("INKROOM_MCP_URL", str(config["mcpUrl"]))
	token := os.Getenv("INKROOM_MCP_TOKEN")
	if token == "" {
		raw, _ := os.ReadFile(tokenPath(mcp))
		token = strings.TrimSpace(string(raw))
	}
	seen := map[string]object{}
	for i, item := range entries {
		if item["saved"] != false || item["source"] != "original_link" {
			continue
		}
		status, _ := item["httpStatus"].(int)
		if status != 403 && status != 404 && status != 410 && !recoverableOriginalError(str(item["errorCode"])) {
			continue
		}
		taskID := str(item["taskId"])
		resultID := str(item["resultId"])
		if taskID == "" || item["index"] == nil || resultID == "" {
			item["reviewStatus"] = "missing_image_identity"
			continue
		}
		if prior, ok := seen[resultID]; ok {
			item["deliveryReview"] = prior
			continue
		}
		if token == "" || mcp == "" {
			item["reviewStatus"] = "authorization_required"
			continue
		}
		payload := object{"jsonrpc": "2.0", "id": id(), "method": "tools/call", "params": object{"name": "report_image_delivery", "arguments": object{"taskId": taskID, "imageIndex": item["index"], "resultId": resultID}}}
		prefix := fmt.Sprintf("%s.delivery-%d-%s", strings.TrimSuffix(record, ".json"), i, id())
		if err := atomic(prefix+".request.json", marshal(payload)); err != nil {
			item["reviewStatus"] = "record_failed"
			continue
		}
		raw, err := request(mcp, payload, map[string]string{"Authorization": "Bearer " + token, "Content-Type": "application/json", "Accept": "application/json, text/event-stream", "MCP-Protocol-Version": "2025-03-26"}, false)
		if len(raw) > 0 {
			if saveErr := atomic(prefix+".response.json", raw); saveErr != nil {
				item["reviewStatus"] = "record_failed"
				continue
			}
		}
		item["reviewResponse"] = prefix + ".response.json"
		if err != nil {
			item["reviewStatus"] = "unconfirmed"
			continue
		}
		review, err := parse(raw)
		if err != nil || review["error"] != nil {
			item["reviewStatus"] = "unconfirmed"
			continue
		}
		result, _ := review["result"].(map[string]any)
		if result["isError"] == true {
			item["reviewStatus"] = "unconfirmed"
			continue
		}
		item["deliveryReview"] = compact(review)
		if recovered := saveMedia(review, output); len(recovered) > 0 {
			item["recoveredMedia"] = recovered
			for _, saved := range recovered {
				if (saved["source"] == "original_link" || saved["alsoDeliveredAs"] == "original_link") && saved["decoded"] == true && saved["resultId"] == resultID {
					item["saved"] = true
					item["recovered"] = true
					for key, val := range saved {
						item[key] = val
					}
				}
			}
		}
		if content, ok := result["structuredContent"].(map[string]any); ok {
			item["reviewStatus"] = content["status"]
		}
		seen[resultID] = object{"response": prefix + ".response.json", "result": compact(review)}
	}
}

// Find resource identities first: JSON object iteration order must not let a
// duplicate URL in structuredContent hide the authoritative resource metadata.
func mediaIdentities(value any) map[string]object {
	out := map[string]object{}
	var walk func(any)
	walk = func(v any) {
		switch n := v.(type) {
		case []any:
			for _, x := range n {
				walk(x)
			}
		case map[string]any:
			if n["type"] == "resource_link" && str(n["uri"]) != "" {
				if id := mediaIdentity(n); str(id["taskId"]) != "" {
					if n["size"] != nil {
						id["expectedBytes"] = n["size"]
					}
					out[str(n["uri"])] = id
				}
			}
			for _, x := range n {
				walk(x)
			}
		}
	}
	walk(value)
	return out
}

// A successful task without an original is also a delivery failure, even when
// there is no URL to attempt. Do not let an empty media list imply success.
func unavailableOriginals(value any) []object {
	out := []object{}
	seen := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		switch n := v.(type) {
		case []any:
			for _, x := range n {
				walk(x)
			}
		case map[string]any:
			if n["type"] == "text" {
				var inner any
				if json.Unmarshal([]byte(str(n["text"])), &inner) == nil {
					walk(inner)
				}
			}
			if manifest, ok := n["resultManifest"].(map[string]any); ok {
				tasks := map[string]string{}
				if inputs, ok := manifest["inputs"].([]any); ok {
					for _, raw := range inputs {
						if input, ok := raw.(map[string]any); ok {
							tasks[fmt.Sprint(input["index"])] = str(input["taskId"])
						}
					}
				}
				if results, ok := manifest["results"].([]any); ok {
					for _, raw := range results {
						item, ok := raw.(map[string]any)
						if !ok || item["status"] != "ready" || item["deliveryStatus"] != "unavailable" {
							continue
						}
						id := str(item["resultId"])
						if id == "" || seen[id] {
							continue
						}
						seen[id] = true
						out = append(out, object{"saved": false, "source": "original_link", "errorCode": "missing_original_link", "error": "completed image has no original link", "taskId": tasks[fmt.Sprint(item["inputIndex"])], "index": item["imageIndex"], "imageIndex": item["imageIndex"], "inputIndex": item["inputIndex"], "resultId": id})
					}
				}
			}
			for k, x := range n {
				if k != "text" {
					walk(x)
				}
			}
		}
	}
	walk(value)
	return out
}

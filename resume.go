package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

func numberInt64(value any) (int64, bool) {
	switch n := value.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		if n >= 0 && n <= maxBytes && math.Trunc(n) == n {
			return int64(n), true
		}
	}
	return 0, false
}
func recoverableOriginalError(code string) bool {
	switch code {
	case "missing_original_link", "download_empty", "download_incomplete", "download_range_invalid", "decode_failed", "download_transport", "download_timeout":
		return true
	}
	return false
}
func defaultSkillDelivery(tool string, input object) {
	switch tool {
	case "generate_image", "batch_generate", "derive_image", "upscale_image", "wait_images":
		if _, present := input["deliveryMode"]; !present {
			input["deliveryMode"] = "links"
		}
	}
}
func responseError(value object) error {
	if value["error"] != nil {
		return errors.New("service returned an error; response retained")
	}
	if result, ok := value["result"].(map[string]any); ok && result["isError"] == true {
		return errors.New("tool rejected request; response retained")
	}
	return nil
}
func continuation(value object) (object, bool) {
	result, _ := value["result"].(map[string]any)
	content, _ := result["structuredContent"].(map[string]any)
	next, _ := content["nextAction"].(map[string]any)
	if next == nil {
		if blocks, ok := result["content"].([]any); ok {
			for _, raw := range blocks {
				block, ok := raw.(map[string]any)
				if !ok || block["type"] != "text" {
					continue
				}
				var inner object
				if json.Unmarshal([]byte(str(block["text"])), &inner) == nil {
					if next, _ = inner["nextAction"].(map[string]any); next != nil {
						break
					}
				}
			}
		}
	}
	// This client never follows a continuation that authorizes new generation.
	if str(next["tool"]) != "wait_images" {
		return nil, false
	}
	args, ok := next["arguments"].(map[string]any)
	if !ok {
		return nil, false
	}
	return object{"name": "wait_images", "arguments": args}, true
}
func resumeResponse(value object, record, output string, config object, maxPolls int, priorMedia ...object) error {
	if maxPolls < 1 || maxPolls > 256 {
		return errors.New("max-polls must be 1..256")
	}
	mcp := env("INKROOM_MCP_URL", str(config["mcpUrl"]))
	token := os.Getenv("INKROOM_MCP_TOKEN")
	if token == "" {
		raw, _ := os.ReadFile(tokenPath(mcp))
		token = strings.TrimSpace(string(raw))
	}
	if token == "" || mcp == "" {
		return errors.New("account authorization required for receipt recovery")
	}
	checkpoint := strings.TrimSuffix(record, ".json") + ".resume.json"
	if chosen := str(config["resumeCheckpoint"]); chosen != "" {
		checkpoint = chosen
	}
	entries := append([]object(nil), priorMedia...)
	positions := map[string]int{}
	for index, item := range entries {
		positions[mediaEntryKey(item)] = index
	}
	for poll := 0; ; poll++ {
		if err := responseError(value); err != nil {
			return err
		}
		media := append(saveMedia(value, output), unavailableOriginals(value)...)
		reviewFailedMedia(media, record, output, config)
		for _, item := range media {
			key := mediaEntryKey(item)
			if index, ok := positions[key]; ok {
				entries[index] = item
			} else {
				positions[key] = len(entries)
				entries = append(entries, item)
			}
		}
		if err := atomic(strings.TrimSuffix(record, ".json")+".manifest.json", marshal(media)); err != nil {
			return err
		}
		next, more := continuation(value)
		complete := !more && responseComplete(value)
		if err := atomic(checkpoint, marshal(object{"latestResponse": record, "media": entries, "nextAction": next, "complete": complete})); err != nil {
			return err
		}
		if !more || poll >= maxPolls {
			fmt.Println(string(marshal(object{"response": record, "checkpoint": checkpoint, "media": entries, "result": compact(value), "continuationPending": more})))
			for _, item := range entries {
				if item["saved"] == false {
					return errors.New("media delivery incomplete; recover the original response; do not regenerate")
				}
			}
			if !more && !complete {
				return errors.New("original task completion is unconfirmed; retain receipt and inspect nextAction")
			}
			if more {
				return errors.New("resume poll limit reached; continue from latestResponse; no new generation submitted")
			}
			return nil
		}
		payload := object{"jsonrpc": "2.0", "id": id(), "method": "tools/call", "params": next}
		newRecord := filepath.Join(output, id()+".response.json")
		if err := atomic(strings.TrimSuffix(newRecord, ".response.json")+".request.json", marshal(payload)); err != nil {
			return err
		}
		raw, err := request(mcp, payload, map[string]string{"Authorization": "Bearer " + token, "Content-Type": "application/json", "Accept": "application/json, text/event-stream", "MCP-Protocol-Version": "2025-03-26"}, false)
		if len(raw) > 0 {
			if e := atomic(newRecord, raw); e != nil {
				return e
			}
		}
		if err != nil {
			return fmt.Errorf("%w; resume checkpoint: %s", err, checkpoint)
		}
		value, err = parse(raw)
		if err != nil {
			return err
		}
		record = newRecord
	}
}

func mediaEntryKey(item object) string {
	key := str(item["source"]) + ":" + str(item["resultId"])
	if str(item["resultId"]) == "" {
		key += str(item["path"]) + str(item["sha256"])
	}
	return key
}

func responseComplete(value object) bool {
	result, _ := value["result"].(map[string]any)
	content, _ := result["structuredContent"].(map[string]any)
	if complete, ok := content["complete"].(bool); ok {
		return complete
	}
	return str(content["status"]) == "completed"
}

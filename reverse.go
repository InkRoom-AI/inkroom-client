package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"flag"
	"image"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Leave room for base64 expansion inside the remote MCP 20 MiB RPC limit.
const reverseImageMaxBytes = 10 << 20
const reverseImageMaxPixels = 40_000_000

func reverse(args []string) error {
	flags := flag.NewFlagSet("reverse", flag.ContinueOnError)
	action := flags.String("action", "preview", "preview, auto, run after approval, or get")
	source := ""
	sourceSeen := false
	flags.Func("image", "one local PNG/JPEG/WebP, public HTTPS/HTTP URL, or data URL; preview only", func(value string) error {
		if sourceSeen {
			return errors.New("one image per reverse preview; --image cannot be repeated")
		}
		source, sourceSeen = value, true
		return nil
	})
	requestID := flags.String("request-id", "", "the existing reverse_prompt preview or result receipt")
	out := flags.String("output", "inkroom-output", "result directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments; one image per reverse preview")
	}
	input := object{"action": *action}
	switch *action {
	case "preview":
		if source == "" || *requestID != "" {
			return errors.New("reverse preview requires one --image and no --request-id")
		}
		encoded, err := reverseImageSource(source)
		if err != nil {
			return err
		}
		input["image"] = encoded
	case "auto", "run", "get":
		if strings.TrimSpace(*requestID) == "" || sourceSeen {
			return errors.New("reverse execution or recovery requires --request-id and no --image")
		}
		input["requestId"] = *requestID
	default:
		return errors.New("reverse action must be preview, auto, run or get")
	}
	output, err := filepath.Abs(*out)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(output, 0700); err != nil {
		return err
	}
	record := filepath.Join(output, id()+".response.json")
	config := settings()
	payload := object{"jsonrpc": "2.0", "id": id(), "method": "tools/call", "params": object{"name": "reverse_prompt", "arguments": input}}
	value, err := accountRequest(payload, record, config)
	if err != nil {
		return err
	}
	// A source URL is input evidence, never a generated original to download.
	return finishResponse(value, record, output, config, false)
}

func reverseImageSource(source string) (string, error) {
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		parsed, err := url.Parse(source)
		if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
			return "", errors.New("image URL must be HTTP(S), without credentials or a fragment")
		}
		return source, nil // Server freezes and validates the public input bytes.
	}
	if strings.HasPrefix(source, "data:") {
		comma := strings.IndexByte(source, ',')
		if comma < 0 || !strings.HasSuffix(source[:comma], ";base64") || len(source[comma+1:]) > base64.StdEncoding.EncodedLen(reverseImageMaxBytes) {
			return "", errors.New("image data URL must contain bounded base64 PNG, JPEG or WebP")
		}
		raw, err := base64.StdEncoding.DecodeString(source[comma+1:])
		if err != nil {
			return "", errors.New("invalid image data URL")
		}
		canonical, err := reverseImageDataURL(raw)
		if err != nil {
			return "", err
		}
		if source[:comma] != canonical[:strings.IndexByte(canonical, ',')] {
			return "", errors.New("image data URL type does not match its decoded bytes")
		}
		return source, nil
	}
	file, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("image input must be a regular file")
	}
	if info.Size() > reverseImageMaxBytes {
		return "", errors.New("image exceeds 10 MiB; resize a copy before preview")
	}
	raw, err := io.ReadAll(io.LimitReader(file, reverseImageMaxBytes+1))
	if err != nil {
		return "", err
	}
	return reverseImageDataURL(raw)
}

func reverseImageDataURL(raw []byte) (string, error) {
	if len(raw) == 0 || len(raw) > reverseImageMaxBytes {
		return "", errors.New("image must be nonempty and at most 10 MiB")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > reverseImageMaxPixels {
		return "", errors.New("image dimensions are invalid or exceed 40 megapixels")
	}
	mime := ""
	switch format {
	case "png":
		mime = "image/png"
	case "jpeg":
		mime = "image/jpeg"
	case "webp":
		mime = "image/webp"
	default:
		return "", errors.New("reverse accepts PNG, JPEG or WebP images")
	}
	if _, _, err = image.Decode(bytes.NewReader(raw)); err != nil {
		return "", errors.New("image is incomplete or cannot be decoded")
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

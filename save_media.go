package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"image"
	"net/http"
	"path/filepath"
	"sync"
)

type mediaCandidate struct {
	Encoded, Link, Source string
	Identity              object
}

// Two bounded workers and one connection pool keep original downloads out of
// Agent orchestration. Result ordering still follows the response identities.
func saveMediaCandidates(candidates []mediaCandidate, output string) []object {
	entries := make([]object, len(candidates))
	client, closeClient := newHTTPClient(true)
	defer closeClient()
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < min(2, len(candidates)); worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				entries[index] = saveMediaCandidate(candidates[index], output, client)
			}
		}()
	}
	for index := range candidates {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	return entries
}
func saveMediaCandidate(candidate mediaCandidate, output string, client *http.Client) object {
	identity := candidate.Identity
	var raw []byte
	var err error
	if candidate.Encoded != "" {
		raw, err = base64.StdEncoding.DecodeString(candidate.Encoded)
	} else {
		if err = endpoint(candidate.Link); err == nil {
			raw, err = downloadOriginalWithClient(context.Background(), client, candidate.Link, output, identity)
		}
	}
	if err == nil && candidate.Source == "original_link" {
		if expected, ok := numberInt64(identity["expectedBytes"]); ok && expected > 0 && int64(len(raw)) != expected {
			err = &mediaFailure{Code: "download_incomplete"}
		}
	}
	var cfg image.Config
	format := ""
	if err == nil {
		cfg, format, err = image.DecodeConfig(bytes.NewReader(raw))
		if err == nil && (cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 100_000_000) {
			err = errors.New("image dimensions exceed limit")
		}
	}
	if err == nil {
		_, _, err = image.Decode(bytes.NewReader(raw))
	}
	if err != nil {
		return mediaFailureEntry(err, candidate.Source, identity)
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	target := filepath.Join(output, digest+"."+format)
	if err := atomic(target, raw); err != nil {
		return mediaFailureEntry(&mediaFailure{Code: "local_write_failed"}, candidate.Source, identity)
	}
	entry := object{"path": target, "sha256": digest, "width": cfg.Width, "height": cfg.Height, "format": format, "bytes": len(raw), "saved": true, "decoded": true, "source": candidate.Source, "mediaKind": "preview"}
	for key, value := range identity {
		entry[key] = value
	}
	if candidate.Source == "original_link" {
		entry["mediaKind"] = "original"
		if err := checkpointOriginal(output, candidate.Link, identity, target, digest); err != nil {
			entry["checkpointStatus"] = "failed"
		}
	}
	return entry
}

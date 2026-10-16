package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type downloadState struct {
	ETag   string `json:"etag,omitempty"`
	Path   string `json:"path,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

func originalKey(identity object, link string) string {
	if resultID := str(identity["resultId"]); resultID != "" {
		return "result:" + resultID
	}
	return "url:" + link
}
func downloadOriginal(link, output string, identity object) ([]byte, error) {
	if err := endpoint(link); err != nil {
		return nil, err
	}
	client, closeClient := newHTTPClient(true)
	defer closeClient()
	return downloadOriginalWithClient(context.Background(), client, link, output, identity)
}

// Retries are GETs of the same original, never paid generation. Strong ETags
// gate partial reuse. Fresh signatures for the same result share the checkpoint.
func downloadOriginalWithClient(ctx context.Context, client *http.Client, link, output string, identity object) ([]byte, error) {
	hash := sha256.Sum256([]byte(originalKey(identity, link)))
	base := filepath.Join(output, ".downloads", hex.EncodeToString(hash[:]))
	if err := os.MkdirAll(filepath.Dir(base), 0700); err != nil {
		return nil, &mediaFailure{Code: "local_write_failed"}
	}
	state := downloadState{}
	if raw, err := os.ReadFile(base + ".json"); err == nil {
		_ = json.Unmarshal(raw, &state)
	}
	if state.Path != "" && state.SHA256 != "" {
		// Do not trust paths outside this output directory from a modified checkpoint.
		if filepath.Dir(state.Path) == output {
			if raw, err := readBoundedMediaFile(state.Path); err == nil {
				sum := sha256.Sum256(raw)
				if hex.EncodeToString(sum[:]) == state.SHA256 {
					return raw, nil
				}
			}
		}
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		offset := int64(0)
		if info, err := os.Stat(base + ".part"); err == nil && info.Size() <= maxBytes && strongETag(state.ETag) {
			offset = info.Size()
		}
		requestCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, link, nil)
		if err != nil {
			cancel()
			return nil, err
		}
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
			req.Header.Set("If-Range", state.ETag)
		}
		res, err := client.Do(req)
		if err != nil {
			cancel()
			last = mediaTransportError(err)
			continue
		}
		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusPartialContent {
			res.Body.Close()
			cancel()
			last = &mediaFailure{Code: "download_http", HTTPStatus: res.StatusCode}
			if res.StatusCode == 416 {
				state.ETag = ""
				continue
			}
			if res.StatusCode == 429 || res.StatusCode >= 500 {
				continue
			}
			return nil, last
		}
		total := res.ContentLength
		if res.StatusCode == http.StatusPartialContent {
			var start, end int64
			if _, err := fmt.Sscanf(res.Header.Get("Content-Range"), "bytes %d-%d/%d", &start, &end, &total); err != nil || start != offset || end < start || total <= end || total > maxBytes || (res.ContentLength >= 0 && res.ContentLength != end-start+1) || (offset > 0 && res.Header.Get("ETag") != state.ETag) {
				res.Body.Close()
				cancel()
				return nil, &mediaFailure{Code: "download_range_invalid"}
			}
		} else {
			offset = 0
		}
		if total > maxBytes {
			res.Body.Close()
			cancel()
			return nil, &mediaFailure{Code: "download_size_limit"}
		}
		state.ETag = res.Header.Get("ETag")
		if err := atomic(base+".json", marshal(state)); err != nil {
			res.Body.Close()
			cancel()
			return nil, &mediaFailure{Code: "local_write_failed"}
		}
		flags := os.O_CREATE | os.O_WRONLY
		if offset == 0 {
			flags |= os.O_TRUNC
		} else {
			flags |= os.O_APPEND
		}
		file, err := os.OpenFile(base+".part", flags, 0600)
		if err != nil {
			res.Body.Close()
			cancel()
			return nil, &mediaFailure{Code: "local_write_failed"}
		}
		n, copyErr := io.Copy(file, io.LimitReader(res.Body, maxBytes-offset+1))
		syncErr := file.Sync()
		closeErr := file.Close()
		res.Body.Close()
		cancel()
		if syncErr != nil || closeErr != nil {
			return nil, &mediaFailure{Code: "local_write_failed"}
		}
		if offset+n > maxBytes {
			return nil, &mediaFailure{Code: "download_size_limit"}
		}
		if copyErr != nil {
			var writeErr *os.PathError
			if errors.As(copyErr, &writeErr) {
				return nil, &mediaFailure{Code: "local_write_failed"}
			}
			last = mediaTransportError(copyErr)
			continue
		}
		if offset+n == 0 {
			state.ETag = ""
			last = &mediaFailure{Code: "download_empty"}
			continue
		}
		if (total >= 0 && offset+n != total) || (res.ContentLength >= 0 && n != res.ContentLength) {
			last = &mediaFailure{Code: "download_incomplete"}
			continue
		}
		return readBoundedMediaFile(base + ".part")
	}
	if last == nil {
		last = &mediaFailure{Code: "download_transport"}
	}
	return nil, last
}
func strongETag(value string) bool {
	return strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") && len(value) > 2
}
func checkpointOriginal(output, link string, identity object, path, digest string) error {
	sum := sha256.Sum256([]byte(originalKey(identity, link)))
	statePath := filepath.Join(output, ".downloads", hex.EncodeToString(sum[:])+".json")
	return atomic(statePath, marshal(downloadState{Path: path, SHA256: digest}))
}

func readBoundedMediaFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if len(data) > maxBytes {
		return nil, &mediaFailure{Code: "download_size_limit"}
	}
	return data, err
}

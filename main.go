// InkRoom's standalone Skill client. All dependencies are linked at build time.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	_ "golang.org/x/image/webp"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

//go:embed SKILL.md
var skill []byte

const version = "1.0.5"
const maxBytes = 100 << 20

type object = map[string]any

func id() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func atomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".inkroom-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
func marshal(v any) []byte { b, _ := json.MarshalIndent(v, "", "  "); return b }
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func stateDir() string {
	h, _ := os.UserHomeDir()
	return env("INKROOM_STATE_DIR", filepath.Join(h, ".inkroom-skill"))
}
func endpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.User != nil || u.Hostname() == "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) {
		return errors.New("use an HTTPS endpoint (HTTP loopback is supported for local testing)")
	}
	return nil
}
func request(raw string, payload any, headers map[string]string, media bool) ([]byte, error) {
	if err := endpoint(raw); err != nil {
		return nil, err
	}
	var body io.Reader
	method := "GET"
	if payload != nil {
		method = "POST"
		body = bytes.NewReader(marshal(payload))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, raw, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client, closeClient := newHTTPClient(media)
	defer closeClient()
	response, err := client.Do(req)
	if err != nil {
		if media {
			return nil, mediaTransportError(err)
		}
		return nil, errors.New("connection failed; retain original request and do not automatically resubmit generation")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		if media {
			return data, mediaTransportError(err)
		}
		return data, errors.New("response interrupted; submission may have completed, do not resubmit")
	}
	if len(data) > maxBytes {
		if media {
			return nil, &mediaFailure{Code: "download_size_limit"}
		}
		return nil, errors.New("response exceeds 100 MiB")
	}
	if media && response.StatusCode >= 400 {
		return nil, &mediaFailure{Code: "download_http", HTTPStatus: response.StatusCode}
	}
	if response.StatusCode >= 400 {
		switch response.StatusCode {
		case http.StatusUnauthorized:
			return data, errors.New("MCP Token 无效或已撤销；请在 InkRoom 重新选择有效 Token")
		case http.StatusForbidden:
			return data, errors.New("MCP Token 没有访问权限；请检查账号和 Token 授权范围")
		default:
			return data, fmt.Errorf("InkRoom 服务返回 HTTP %d；原始响应已保存", response.StatusCode)
		}
	}
	return data, nil
}
func newHTTPClient(media bool) (*http.Client, func()) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if media { // Resolve and dial only public addresses; never forward authentication.
		transport.Proxy = nil
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, errors.New("media host has no addresses")
			}
			for _, ip := range ips {
				if !ip.IP.IsGlobalUnicast() || ip.IP.IsPrivate() || ip.IP.IsLoopback() || ip.IP.IsLinkLocalUnicast() {
					return nil, errors.New("private media address refused")
				}
			}
			return (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		}
	}
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("redirect refused; verify endpoint or refresh media link")
	}}
	return client, transport.CloseIdleConnections
}
func parse(raw []byte) (object, error) {
	var result object
	if err := json.Unmarshal(raw, &result); err == nil && result != nil {
		return result, nil
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	for _, event := range strings.Split(text, "\n\n") {
		var parts []string
		for _, line := range strings.Split(event, "\n") {
			if strings.HasPrefix(line, "data:") {
				parts = append(parts, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		if json.Unmarshal([]byte(strings.Join(parts, "\n")), &result) == nil && result != nil && (result["result"] != nil || result["error"] != nil) {
			return result, nil
		}
	}
	return nil, errors.New("no JSON result; raw response retained")
}
func str(v any) string { s, _ := v.(string); return s }
func compact(v any) any {
	switch n := v.(type) {
	case []any:
		out := []any{}
		for _, x := range n {
			if m, ok := x.(map[string]any); ok && m["type"] == "image" {
				continue
			}
			out = append(out, compact(x))
		}
		return out
	case map[string]any:
		out := object{}
		for k, x := range n {
			if k == "b64_json" || k == "token" {
				continue
			}
			if k == "text" && n["type"] == "text" {
				var inner any
				if json.Unmarshal([]byte(str(x)), &inner) == nil {
					out[k] = string(marshal(compact(inner)))
					continue
				}
			}
			out[k] = compact(x)
		}
		return out
	}
	return v
}
func saveMedia(value any, output string) []object {
	identities := mediaIdentities(value)
	candidates := []mediaCandidate{}
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
			encoded := str(n["b64_json"])
			if n["type"] == "image" {
				encoded = str(n["data"])
			}
			link := ""
			if n["type"] == "resource_link" {
				link = str(n["uri"])
			} else if str(n["url"]) != "" && (len(n) == 1 || n["width"] != nil || n["mimeType"] != nil || n["revised_prompt"] != nil) {
				link = str(n["url"])
			}
			if encoded != "" || link != "" {
				source := "original_link"
				identity := identities[link]
				key := source + ":" + originalKey(identity, link)
				if encoded != "" {
					source = "inline"
					identity = mediaIdentity(n)
					key = "inline:" + str(identity["resultId"]) + ":" + encoded
				}
				if !seen[key] {
					seen[key] = true
					candidates = append(candidates, mediaCandidate{encoded, link, source, identity})
				}
			}
			for key, x := range n {
				if key != "b64_json" && key != "text" {
					walk(x)
				}
			}
		}
	}
	walk(value)
	return saveMediaCandidates(candidates, output)
}

func settings() object {
	exe, _ := os.Executable()
	b, _ := os.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(exe)), "settings.json"))
	v := object{}
	_ = json.Unmarshal(b, &v)
	return v
}
func install(args []string) error {
	flags := flag.NewFlagSet("install", flag.ContinueOnError)
	dir := flags.String("skills-dir", "", "host's actual user-level skills directory")
	origin := flags.String("origin", "", "website origin; discovers service endpoints automatically")
	api := flags.String("api-origin", "", "InkRoom API origin")
	mcp := flags.String("mcp-url", "", "InkRoom MCP endpoint")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		if codexHome := os.Getenv("CODEX_HOME"); codexHome != "" {
			*dir = filepath.Join(codexHome, "skills")
		} else {
			home, _ := os.UserHomeDir()
			*dir = filepath.Join(home, ".agents", "skills")
		}
	}
	if *origin != "" {
		base := strings.TrimRight(*origin, "/")
		raw, err := request(base+"/api/settings", nil, nil, false)
		if err != nil {
			return err
		}
		response, err := parse(raw)
		if err != nil {
			return err
		}
		data, _ := response["data"].(map[string]any)
		endpoints, _ := data["endpoints"].(map[string]any)
		if response["code"] != float64(0) || data == nil {
			return errors.New("cannot discover service endpoints; installation did not continue")
		}
		*api = str(endpoints["apiOrigin"])
		*mcp = str(endpoints["mcpUrl"])
		if *api == "" {
			*api = base
		}
		if *mcp == "" {
			*mcp = base + "/api/mcp"
		}
	}

	if err := endpoint(*api); err != nil {
		return err
	}
	if err := endpoint(*mcp); err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	root := strings.Replace(*dir, "~/", home+string(os.PathSeparator), 1)
	target := filepath.Join(root, "inkroom")
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		return errors.New("InkRoom already exists; preserved. Review updates separately")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(root, ".inkroom-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	binary, err := os.ReadFile(exe)
	if err != nil {
		return err
	}
	name := "inkroom"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for path, data := range map[string][]byte{"SKILL.md": skill, "settings.json": marshal(object{"apiOrigin": *api, "mcpUrl": *mcp}), filepath.Join("scripts", name): binary} {
		if err := atomic(filepath.Join(stage, path), data); err != nil {
			return err
		}
	}
	if err := os.Chmod(filepath.Join(stage, "scripts", name), 0755); err != nil {
		return err
	}
	if err := os.Rename(stage, target); err != nil {
		return err
	}
	fmt.Println(string(marshal(object{"installed": target, "platform": runtime.GOOS + "-" + runtime.GOARCH, "generationSubmitted": false})))
	return nil
}
func tokenPath(mcp string) string {
	sum := sha256.Sum256([]byte(mcp))
	return filepath.Join(stateDir(), "tokens", hex.EncodeToString(sum[:])+".token")
}
func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Println("InkRoom Skill " + version + " (" + runtime.GOOS + "/" + runtime.GOARCH + ")\nCommands: install --origin URL [--skills-dir DIR] | authorize (Token from stdin) | tools | call --tool NAME --arguments FILE | recover --response FILE | resume --response FILE [--max-polls N]\nUse --output DIR after a command. Account: INKROOM_MCP_TOKEN environment variable. No runtime dependencies.")
		return nil
	}
	if args[0] == "authorize" {
		config := settings()
		mcp := env("INKROOM_MCP_URL", str(config["mcpUrl"]))
		if err := endpoint(mcp); err != nil {
			return err
		}
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 16385))
		if err != nil {
			return err
		}
		token := strings.TrimSpace(string(raw))
		if token == "" || len(raw) > 16384 {
			return errors.New("empty or oversized Token")
		}
		payload := object{"jsonrpc": "2.0", "id": id(), "method": "tools/list"}
		response, err := request(mcp, payload, map[string]string{"Authorization": "Bearer " + token, "Content-Type": "application/json", "Accept": "application/json, text/event-stream", "MCP-Protocol-Version": "2025-03-26"}, false)
		if err != nil {
			return err
		}
		result, err := parse(response)
		if err != nil || result["error"] != nil || result["result"] == nil {
			return errors.New("无法验证 MCP Token；未保存授权")
		}
		if err = atomic(tokenPath(mcp), []byte(token)); err != nil {
			return err
		}
		fmt.Println("Account authorization stored locally; spending policy unchanged.")
		return nil
	}
	if args[0] == "install" {
		return install(args[1:])
	}
	command := args[0]
	switch command {
	case "guest-access", "guest-generate":
		return errors.New("Skill 仅限注册用户，请登录并使用有效 MCP Token")
	case "tools", "call", "recover", "resume":
	default:
		return errors.New("unknown command; run --help")
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	out := flags.String("output", "inkroom-output", "result directory")
	tool := flags.String("tool", "", "MCP tool name")
	arguments := flags.String("arguments", "", "JSON arguments file")
	responseFile := flags.String("response", "", "original response file")
	maxPolls := flags.Int("max-polls", 32, "maximum read-only continuation calls for resume")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
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
	var value object
	var priorMedia []object
	if command == "recover" || command == "resume" {
		record, err = filepath.Abs(*responseFile)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(record)
		if err != nil {
			return err
		}
		value, err = parse(raw)
		if err != nil {
			return err
		}

		if command == "resume" && str(value["latestResponse"]) != "" {
			config["resumeCheckpoint"] = record
			if saved, ok := value["media"].([]any); ok {
				for _, raw := range saved {
					if entry, ok := raw.(map[string]any); ok {
						priorMedia = append(priorMedia, entry)
					}
				}
			}
			latest, absErr := filepath.Abs(str(value["latestResponse"]))
			if absErr != nil || filepath.Dir(latest) != output {
				return errors.New("resume response must belong to the selected output directory")
			}
			record = latest
			raw, readErr := os.ReadFile(record)
			if readErr != nil {
				return readErr
			}
			value, err = parse(raw)
			if err != nil {
				return err
			}
		}

	} else {
		mcp := env("INKROOM_MCP_URL", str(config["mcpUrl"]))
		token := os.Getenv("INKROOM_MCP_TOKEN")
		if token == "" {
			raw, _ := os.ReadFile(tokenPath(mcp))
			token = strings.TrimSpace(string(raw))
		}
		if token == "" {
			return errors.New("account authorization required; no automatic fallback to guest")
		}
		if mcp == "" {
			return errors.New("set INKROOM_MCP_URL or use install")
		}
		payload := object{"jsonrpc": "2.0", "id": id(), "method": "tools/list"}
		if command == "call" {
			raw, err := os.ReadFile(*arguments)
			if err != nil {
				return err
			}
			var input object
			if err = json.Unmarshal(raw, &input); err != nil || input == nil {
				return errors.New("arguments must be a JSON object")
			}
			if *tool == "" {
				return errors.New("--tool is required")
			}
			defaultSkillDelivery(*tool, input)
			payload["method"] = "tools/call"
			payload["params"] = object{"name": *tool, "arguments": input}
		}
		requestRecord := strings.TrimSuffix(record, ".response.json") + ".request.json"
		if err = atomic(requestRecord, marshal(payload)); err != nil {
			return err
		}
		raw, err := request(mcp, payload, map[string]string{"Authorization": "Bearer " + token, "Content-Type": "application/json", "Accept": "application/json, text/event-stream", "MCP-Protocol-Version": "2025-03-26"}, false)
		if len(raw) > 0 {
			if saveErr := atomic(record, raw); saveErr != nil {
				return saveErr
			}
		}
		if err != nil {
			if len(raw) > 0 {
				return fmt.Errorf("%w；请求记录：%s；响应记录：%s", err, requestRecord, record)
			}
			return fmt.Errorf("%w；请求记录：%s", err, requestRecord)
		}
		value, err = parse(raw)
		if err != nil {
			return err
		}
	}
	if command == "resume" {
		return resumeResponse(value, record, output, config, *maxPolls, priorMedia...)
	}
	media := append(saveMedia(value, output), unavailableOriginals(value)...)
	reviewFailedMedia(media, record, output, config)
	if err = atomic(strings.TrimSuffix(record, ".json")+".manifest.json", marshal(media)); err != nil {
		return err
	}
	fmt.Println(string(marshal(object{"response": record, "media": media, "result": compact(value)})))
	if value["error"] != nil {
		return errors.New("service returned an error; response retained")
	}
	if code, ok := value["code"].(float64); ok && code != 0 {
		return errors.New("service rejected request; response retained")
	}
	if result, ok := value["result"].(map[string]any); ok && result["isError"] == true {
		return errors.New("tool rejected request; response retained")
	}
	for _, item := range media {
		if item["saved"] == false {
			return errors.New("media delivery incomplete; inspect the saved manifest and delivery review; do not regenerate")
		}
	}
	return nil
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

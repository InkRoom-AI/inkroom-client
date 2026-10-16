# InkRoom client

Official standalone client and Skill instructions for [InkRoom](https://inkroom.ai).

High-quality AI images. Fast batch workflows. Web / API / MCP / Skill.

## Scope

Only the user-installed client and Skill instructions are included. Image generation runs on InkRoom Remote MCP. Server source, accounts, provider credentials and business data are excluded.

## Build

Requires Go 1.25.13 or newer.

```sh
go mod download
go test ./...
go build -o inkroom .
./inkroom --help
```

For ready-to-use installation, sign in at https://inkroom.ai and use your account's Skill setup instructions. Never publish Tokens or authorization setup prompts. Generation requires an InkRoom account and MCP Token; charges follow preview, authorization policy and actual account pricing.

## Skill

See [SKILL.md](SKILL.md). Supports discovery, cost previews, confirmed generation, batches, receipt recovery and original-image delivery. This client is not an MCP server.

## Official website

https://inkroom.ai

## Source availability

Published for inspection and integration with InkRoom. No separate open-source license is granted at this stage. Third-party dependencies retain their own licenses.

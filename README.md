# InkRoom client

## MCP, Skills and public materials

Connect through [Remote MCP](MCP-GUIDE.md), explore [image workflows and directory copy](MARKETING.md), or use the standalone client below. The public repository is maintained by the InkRoomAI brand account in the InkRoom-AI organization.


**[Website: inkroom.ai](https://inkroom.ai)** · [简体中文](README.zh-CN.md) · [Skill guide](SKILL.md)

High-quality AI images. Efficient batches. Web / API / MCP / Skill.

Official standalone client and Skill instructions for InkRoom. Use the website for everyday image creation, the API for application integration and batches, or MCP and Skill for AI assistant workflows. Typical uses include product visuals, anime artwork and posters.

## Install in WorkBuddy

[Open InkRoom Image Creation in WorkBuddy](https://www.workbuddy.cn/share/asset?shareCode=OOG16KkXbM3k5ghACZI5x1&sig=L8-dKfDjnKRGnew_)

Use this marketplace Skill if you work in WorkBuddy. The marketplace package is maintained separately from this Go client. Open the page in WorkBuddy to install it, then follow its account authorization instructions. An InkRoom account and a valid MCP Token are required for generation; installing the Skill does not grant free image generation.

## Install

For a ready-to-use installation, sign in at [inkroom.ai](https://inkroom.ai) and follow your account's Skill setup instructions. Use the current platform-specific download and authorization details shown there.

To build from source, install Go 1.25.13 or newer:

```sh
git clone https://github.com/InkRoom-AI/inkroom-client.git
cd inkroom-client
go mod download
go test ./...
go build -o inkroom .
./inkroom --help
```

On Windows, build as `inkroom.exe` and use that executable in the commands below. Prebuilt standalone executables do not require Go, Python or Node.js at runtime.

## Quick start

Complete account setup first. A valid MCP Token is required. Use the account setup instructions; authorization accepts the Token through standard input, or your secure execution environment can provide `INKROOM_MCP_TOKEN`. Never commit Tokens or place them in process arguments, examples or screenshots.

Discover the currently available tools and their schemas:

```sh
./inkroom tools --output ./inkroom-output
```

Call a tool using a JSON argument file:

```sh
./inkroom call --tool TOOL_NAME --arguments ./arguments.json --output ./inkroom-output
```

Replace `TOOL_NAME` with a discovered name and create `arguments.json` according to that tool's current schema. This is a command template, not a ready-made generation request. Do not guess model IDs or hardcode prices.

## Example requests for your AI assistant

After installing the Skill, try:
- “Create four product visuals from this reference image. Keep the product appearance consistent. Show the actual model and estimated cost first.”
- “Create three poster variations using this prompt. Preserve the supplied text.”
- “Resume my previous batch and download the original images. Do not generate again.”

These are natural-language examples, not tool payloads. Available models, image counts, reference support and prices depend on the current tools and account. Preview costs first and follow the account's authorization policy; never bypass spending limits or insufficient balance.

## Download and recover

Keep the original response files and receipts. A failed download does not require another paid generation.

```sh
# Retry saving media from an existing response
./inkroom recover --response ./inkroom-output/RESPONSE_FILE.response.json --output ./inkroom-output

# Follow read-only waits for the existing task and save completed results
./inkroom resume --response ./inkroom-output/RESPONSE_FILE.response.json --output ./inkroom-output
```

Replace the response filename with your actual saved file. See [SKILL.md](SKILL.md) for previews, authorization, batches, recovery and delivery rules.

## Scope and source availability

Only the user-installed client and Skill instructions are included. Image generation runs on InkRoom Remote MCP. This client is not an MCP server or a public API SDK. Server source, accounts, provider credentials and business data are excluded.

Published for inspection and integration with InkRoom. **No separate open-source license is granted at this stage.** Third-party dependencies retain their own licenses.

[Official website](https://inkroom.ai) · [GitHub organization](https://github.com/InkRoom-AI)



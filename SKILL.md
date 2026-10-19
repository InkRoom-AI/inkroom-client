---
name: inkroom
description: Generate and edit images with InkRoom using a registered account and MCP Token; supports batch generation, processing, models, prompts, usage and receipt recovery.
metadata:
  version: "1.0.3"
---

# InkRoom

[InkRoom official website](https://inkroom.ai/) · [Skill setup and documentation](https://inkroom.ai/skill-server) · [Remote MCP documentation](https://inkroom.ai/mcp-server)

Use the standalone executable at `scripts/inkroom` (`scripts/inkroom.exe` on Windows). A prebuilt executable needs no Python, Node or Go runtime. Run `--help` for syntax. Never substitute another model, rewrite the supplied prompt, change references or increase count without permission.

## Install the client

Check whether the executable exists before calling it. Platform-specific packages downloaded from InkRoom include it; marketplace imports of the GitHub repository may contain only Go source files.

For a source-only package containing `go.mod` and `main.go`, build the native client from this skill directory with Go 1.25.13 or newer:

```bash
go build -o scripts/inkroom .
./scripts/inkroom --help
```

On Windows use `go build -o scripts/inkroom.exe .` followed by `scripts\\inkroom.exe --help`. Build for the current host; do not run a binary for another operating system or architecture. If Go is unavailable or the build fails, ask the user to sign in at https://inkroom.ai/skill-server and copy the platform-specific installation and account setup instructions. Do not claim installation succeeded until the executable's `--help` succeeds. Building or checking help does not generate images or require a Token.

## Access

Skill is account-only. Require a valid MCP Token; do not call website guest endpoints or collect hardware identifiers. Without authorization, ask the user to sign in to InkRoom and copy their account setup prompt.

For account capabilities use the locally saved account authorization or `INKROOM_MCP_TOKEN`. The one-paste account setup sends the supplied Token to `authorize` via standard input; never as a process argument. `authorize` verifies the Token through a read-only tool discovery request before saving it locally. If verification fails, ask the user to select a valid Token; do not retry paid generation or fall back to guest. Never put credentials into this skill, command arguments, generated artifacts or shared messages. Account mode uses the existing MCP service, scopes, spending policy and billing.

## Grok Bot

Use the same standalone client on Grok Bot's shared cloud computer. Confirm the actual architecture and installation path. Keep these credential-free instructions in the private skill library and verify discovery in the `/` menu. Receive the Key through a supported secure credential request and `authorize` standard input; if secure transfer is unavailable, report the blocked step. Never put the Key in ordinary chat or model-visible tool arguments. Save originals in a project directory under `/workspace`, then inspect and display the actual files. Do not create routines during setup. See [Grok Bot Skills](https://docs.x.ai/grok-bot/skills-routines-and-automations).

## Account capabilities

Call `tools` to discover current schemas; `call --tool TOOL --arguments FILE` accepts a JSON object. Use help or list_models only when needed. Tools include account, prompts, generate_image, derive_image, upscale_image, batch_generate, generation_status and wait_images, according to current server permissions.

Use the current public model IDs returned by the server for new model selection, comparisons and examples. These are the same configured names used by the website, public API and MCP. Never replace them with an upstream technical ID or append an `@channel` suffix. When the user explicitly supplies an original model name or a historical alias, preserve it unchanged: the server resolves compatible old names to the same model and pricing. If no model is specified, omit it and let the server choose; show the actual public model name and quote from the preview instead of hardcoding a client default or price.

Preview generation first. Preserve the returned requestId or batchId, then call auto. Accounts without a saved spending policy default to unlimited automatic execution; existing saved policies, including disabled or limited policies, remain unchanged. Ask for approval of the displayed maximum before run only if the server returns confirmation_required. Users can set account limits in InkRoom MCP settings. At first paid use, if the policy already read during setup or `spending_policy get` has revision 0, remind the user once that optional rolling 5-hour, daily and weekly (Monday reset) limits can prevent rapid credit use. Continue auto without waiting; do not enable or save limits, and do not query the policy before every task just for this reminder. Unlimited account authorization still requires sufficient balance and obeys Token hard limits and permissions. Never bypass a saved limit by splitting requests. Submit independent jobs as one batch; the server manages concurrency. For edits preserve reference purposes and exact original references.

## Delivery and recovery

Each response is saved atomically with a unique filename before interpretation. The client saves inline images and original resource links locally and writes a manifest of paths and SHA-256 hashes. Each success/failure retains `resultId`, `taskId`, zero-based `imageIndex` and `inputIndex` when provided. User-facing image numbers are one-based positions in the original request, not completion order. Different result identities stay separate even when bytes/hash match. Inline media is a preview and is not evidence that the original was saved. It decodes PNG, JPEG and WebP and records actual dimensions. Inspect saved images with the host image tools to verify semantic content and display. Display the saved images using the host's supported image renderer.

Generation and wait calls default to `deliveryMode: "links"` when the caller does not choose a mode. Keep an explicitly chosen mode. Link pages contain at most 32 originals; native pages contain at most 6 previews.

Follow returned nextAction arguments unchanged, including cursor, using another call. For a long batch use `resume --response FILE --output DIR`: it follows only read-only `wait_images`, saves every request/response before parsing, and writes a `.resume.json` checkpoint with all original identities. To continue after interruption or the bounded poll limit, pass that checkpoint to `resume` with the same output directory. Never follow a continuation that starts generation. Resume original receipts with wait_images; omit cursor to retrieve completed media again when needed. Do not regenerate because output was truncated, a local download failed, or the client exited. Use `recover --response FILE` to retry local media saving without generating. A saved response is not proof of every expected image: reconcile requested count, task status, settlement and saved originals. Report missing or failed items explicitly. Download failures must not trigger paid generation.

## Original download failures

Client version 1.0.5 records `errorCode`, `httpStatus`, and the original image identity in the media manifest. A failed media save exits nonzero, even when generation succeeded. `download_timeout`, `download_transport`, `decode_failed`, and `local_write_failed` are different outcomes; do not infer a server refund from them.

Original downloads use two bounded workers and a shared connection pool. Transport retries are GET-only, at most three attempts, with private `.part` checkpoints and strong ETag / If-Range checks. An unsupported range falls back to a complete GET. Empty bodies, wrong ranges and incomplete lengths fail; decoded originals are atomically saved. A working verified original is reused across fresh URLs using its result identity.

For a completed image whose result manifest says delivery unavailable, an original resource link returning HTTP 403, 404 or 410, or an original download that remains empty, truncated, undecodable or transport-failed after its bounded retry, the client automatically calls `report_image_delivery` with the original `taskId`, `imageIndex`, and `resultId`, saving that request and response atomically. The server checks its own live original URL and ownership. A server-confirmed 403/404/410 enters an idempotent refund of the actual retained charge for that image; previews, other images and previously refunded amounts are not charged/refunded again. `refund_pending` means queued, not credited. Only `refunded` plus settlement confirms the result. If the server returns a working original, the client tries saving that original without generation.

For other Agent clients, use the same tool and original identity to request verification. Never submit a URL, amount or client status code as refund proof. Timeouts, temporary errors, normal retention expiry, user deletion and local disk errors do not automatically refund. If the server cannot establish ownership, unique media or billing, preserve the receipt and report unconfirmed. Resume original tasks with `wait_images`; never regenerate to repair a download. Older server versions may not expose the review tool: preserve the failed download evidence and report that verification is unavailable.

## Funds and retention

Batch previews return `funding`: the actual applicable account credits, this batch's exact required amount and shortfall are separate from automatic spending authorization. `preflight_not_reserved` does not freeze money. If funding is insufficient, show the gap; do not reduce count, split a batch, raise authorization, or submit until the user has funded the account or explicitly changed the request. A fresh submission rechecks funding; existing receipts restore the original work regardless of current balance. Original quoted maximum stays fixed; actual charged/refunded/net amounts and pending/unknown states are separate.

For new immediate MCP/Skill results, the original object and original signed URL share a fixed 72-hour expiration measured from successful storage verification. Reading a receipt does not extend that deadline. Old results keep their original deadline. Read `retainedUntil` and `linkExpiresAt` from the response; private retry-input snapshots retain their separate one-hour window. Link/object expiration after the promised window is not a service refund proof.

A local `insufficient_balance` failure may restore only the original failed inputs for a new preview after top-up. It is not automatically retryable and never authorizes a new charge. Failed-image counts and their own billing evidence must match; never reuse the parent's refund amount for each failed image.

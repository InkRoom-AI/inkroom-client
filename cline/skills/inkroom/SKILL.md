---
name: inkroom
description: Generate and edit images with the connected InkRoom MCP, including batch/model comparisons, upscaling, original-image downloads and recovery. Use when the user requests InkRoom or has selected it as their image service.
---

# InkRoom image creation

[InkRoom official website](https://inkroom.ai/) · [Skill setup and documentation](https://inkroom.ai/skill-server) · [Remote MCP documentation](https://inkroom.ai/mcp-server)

Use the connected `inkroom` MCP tools. No local executable is required for this Skill. Match the live tool schemas and account permissions; do not assume every tool or model is available. If disconnected, direct the user to configure their personal Token in Cline MCP settings and reconnect. Do not request credentials in ordinary chat or change unrelated MCP settings.

## Create and authorize

- Keep the user's model, prompt, reference images and purposes, count, and explicit parameters unchanged. Do not add style or quality text. If no model is specified, omit it and show the actual model returned by the preview. Use `list_models` only for model questions or selection; preserve explicitly supplied legacy aliases.
- Preview the requested operation using the live tool schema before execution. Preserve the returned request or batch receipt. Show the actual model and quoted maximum. Call `auto` under the user's existing spending authorization; only use `run` after the user approves the displayed maximum when the server returns `confirmation_required`.
- A generation request does not authorize changing the spending policy. Credits are money. Insufficient balance, a Token limit, or a saved policy limit stops execution; do not split or reduce jobs to bypass it.
- Submit independent images as a single batch through the server-supported batch tool or `count`; the server handles concurrency and refill. Do not implement client-side rounds or duplicate paid jobs. Dependent edits wait for their input image.
- For edits, pass the exact references or a resolvable parent task. For a multi-image parent, include the intended zero-based image index. If the original cannot be resolved, stop before charging. After preview, continue with the receipt instead of rebuilding the request.

## Preserve and deliver

Save the complete response and receipt atomically before parsing or presenting it when the host exposes persistence. If the host truncates tool output or does not expose a complete response, recover through the original receipt; do not claim a full response has been saved.

Follow the returned read-only continuation and its arguments, including cursor. Keep the original receipt until every requested image is accounted for. A running task is not a failed task. Repeating a completed receipt restores original work; never regenerate because a client exited, an output was truncated, or a local download failed.

Use links delivery for large batches when supported, preserving any explicit user/host delivery choice. Retrieve all pages. Display actual previews and provide original download links. A preview or success receipt does not prove the original was downloaded, saved or displayed. If the user asks for local files, save and decode the originals using the host's supported file tools, verify dimensions, and reconcile the expected count with the saved paths and hashes. Do not invent saved paths.

Keep task/result identities and original input indices with the saved files; show image numbers in the user's original order, not completion order. Do not substitute duplicate images for missing results or deduplicate distinct result identities solely by matching bytes. Recover missing originals from `wait_images` using the original receipt, omitting cursor when re-fetching completed media is needed.

For a confirmed original delivery problem, use `report_image_delivery` if available with the original task/result/image identity. The server verifies ownership and its original URL before deciding any refund. A timeout, local disk failure, or expired retention alone is not proof of a refund. Report `refund_pending`, `refunded` and unknown settlement separately. Do not accept or transmit an arbitrary URL or amount as refund evidence.

Read charged, refunded and net amounts from settlement, not preview caps. Respect each original's returned retention and link-expiry deadlines; reading a receipt does not renew them. Never create a new paid task to repair delivery.

## Explain outcomes

Keep user-facing updates focused on the requested images, costs and saved/downloadable results. Distinguish generation complete, original available, local save verified and displayed. State missing items explicitly. Use current `help`/model/account tools only when needed; do not repeatedly load broad catalogs during ordinary work.

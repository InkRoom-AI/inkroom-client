# InkRoom for Cline

Official InkRoom remote MCP and companion image-creation Skill. Generate/edit images, compare models in batches, process images and recover original results through the existing InkRoom service. This directory contains installation guidance and a Skill, not the hosted MCP server implementation. No local server build, Go binary, startup script or paid model is needed to configure the connection.

## Requirements

- Cline with remote Streamable HTTP support.
- Your own InkRoom account and personal MCP Token from https://inkroom.ai/mcp-server .
- Image generation uses your InkRoom credits; credits are money. Installing the connection/Skill is free. Tool discovery does not generate an image.

## Connect MCP

Open Cline → MCP Servers → Configure → Configure MCP Servers. Merge the `inkroom` entry below into the existing `mcpServers` object. Preserve all unrelated servers and any existing InkRoom account configuration. If `inkroom` already exists, inspect it and reuse it instead of creating a duplicate. Replace the placeholder privately with the user's own Token; never ask them to paste it into ordinary chat or commit it to Git.

```json
{
  "mcpServers": {
    "inkroom": {
      "type": "streamableHttp",
      "url": "https://mcp.inkroom.ai",
      "headers": {
        "Authorization": "Bearer YOUR_INKROOM_MCP_TOKEN"
      },
      "disabled": false,
      "autoApprove": []
    }
  }
}
```

Set `type` explicitly to `streamableHttp`; omitting it can select legacy SSE. Bearer authentication is required; InkRoom does not offer OAuth sign-in. The `autoApprove` list intentionally stays empty. Never auto-approve generation or alter a user's spending policy as an installation step.

Reconnect and verify InkRoom tools are discovered. Call the read-only `help` tool once to verify authenticated access. Do not invoke generate/batch/process or change authorization just to test the installation. A 401 means credentials are missing or invalid, not successful setup. For network failures, retain the original configuration and report the actual error.

## Install the companion Skill

Install only this directory's `skills/inkroom` through Cline's supported Skill installation flow. It guides the connected MCP and needs no standalone executable. Do not import the repository-root Go-client Skill by mistake. Verify that the installed `SKILL.md` matches this directory and that Cline can discover it before reporting success.

Source: https://github.com/InkRoom-AI/inkroom-client/tree/main/cline/skills/inkroom

Try: “Use InkRoom to create one cafe poster. Show the model and cost first.” Respect the account's saved authorization. If the server returns `confirmation_required`, obtain approval for its quoted maximum before execution. Do not split work to bypass limits.

For incomplete or failed image delivery, recover the original task and all result pages; do not regenerate or pay again. A generated receipt, working original link, decoded local file and preview display are different outcomes.

## Privacy, support and license

Prompts and references are sent to InkRoom when the user invokes its tools. Public files contain no credentials. Privacy: https://inkroom.ai/privacy-policy . Website and support: https://inkroom.ai . MCP guide/account setup: https://inkroom.ai/mcp-server .

This directory's companion Skill and installation documentation are MIT-licensed; see LICENSE. This does not license the hosted service, the existing standalone Go client, or the repository's other files. The Skill adapts the MIT-licensed InkRoom Cursor Skill.

# InkRoom for Cursor

Official InkRoom plugin: a remote MCP connection and an image-creation Skill in one install. The Skill guides the connected MCP; it does not require the standalone Go client.

## Connect

1. Register/sign in at https://inkroom.ai and create a personal MCP Token at https://inkroom.ai/mcp-server.
2. Install the InkRoom plugin. In Cursor's plugin configuration, set `INKROOM_MCP_TOKEN` to your personal Token without the `Bearer ` prefix. The variable is required; the repository contains only its placeholder. Cursor manages plugin variables in Plugins → Configure; available configuration surfaces depend on the Cursor account and version. If your install does not expose the field, use the documented manual fallback below rather than pasting a Key into chat.
3. Reconnect the `inkroom` MCP and verify its tools are visible. The Skill should appear as `inkroom`. A successful install is not proof of a completed image generation.

The transport is Streamable HTTP at `https://mcp.inkroom.ai`. Each user supplies their own account Token. The plugin does not grant free credits or share a publisher account.

## Manual fallback

If your Cursor version cannot configure plugin variables, add the same remote server through Cursor's MCP settings using your own Token privately. Keep any existing `inkroom` connection and unrelated connectors; do not create duplicate connections. Load this plugin's Skill separately through Cursor's supported local Skill/plugin mechanism. Do not claim the one-click plugin connection succeeded until it is connected.

```json
{
  "mcpServers": {
    "inkroom": {
      "url": "https://mcp.inkroom.ai",
      "headers": { "Authorization": "Bearer YOUR_INKROOM_MCP_TOKEN" }
    }
  }
}
```

## Try it

- “Use InkRoom to make one cafe poster. Show the model and estimated cost first.”
- “Create a batch of product visuals from this reference, preserving the product.”
- “Recover my previous InkRoom batch and save its original images. Do not generate again.”

Model support, references, tools, permissions and prices are returned by InkRoom. Preview first; execution follows the account's saved authorization. A `confirmation_required` result requires approval of the maximum before `run`. Never bypass limits by splitting jobs.

Preserve original receipts, follow all result pages, and download/verify originals when local files are requested. A failed download does not authorize another paid generation. Refunds depend on server-verified original delivery and settlement.

## Distribution and privacy

The plugin contains only public configuration, image-workflow instructions and a brand logo. It uploads prompts/references to InkRoom only when the user invokes its tools. No startup hooks, background jobs, telemetry scripts or embedded credentials are included. Data handling: https://inkroom.ai/privacy-policy . Documentation: https://inkroom.ai/mcp-server . Brand source: https://github.com/InkRoom-AI/inkroom-client .

Plugin distribution version: 1.0.0. Remote MCP rollout and standalone-client versions are independent. The new plugin configuration, Skill and documentation use MIT; the unmodified brand logo may be displayed only with this plugin. See LICENSE and LICENSE-SCOPE.md for exact scope. The standalone Go client and server licensing remain unchanged.

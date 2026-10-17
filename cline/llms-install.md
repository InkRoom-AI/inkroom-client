# Cline installation contract

Read README.md in this directory. Connect only the remote MCP at `https://mcp.inkroom.ai` with explicit `streamableHttp` transport, a private user-owned Bearer Token, and an empty autoApprove list. Merge without overwriting other servers or existing InkRoom credentials. There is no local MCP server process to clone/build/run.

The companion Skill is `skills/inkroom/SKILL.md` in this directory, not the Go-client Skill at repository root. Install only that selected Skill. Verify tool discovery and one read-only help call before confirming connected; do not generate images or change spending policy during installation. Never hardcode models, prices, actual credentials or made-up CLI flags. Follow the installed Cline version's supported configuration UI/CLI.

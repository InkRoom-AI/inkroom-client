# InkRoom — Remote MCP & Agent Skills

[English](#connect-to-inkroom) · [简体中文](#中文接入指南) · [Website](https://inkroom.ai/en) · [MCP guide](https://inkroom.ai/en/solutions/mcp-server) · [Agent Skills](https://inkroom.ai/en/solutions/agent-skills)

InkRoom connects AI assistants to image generation, reference-based editing, batches, upscaling and original-image delivery. Common workflows include ecommerce product photography, white-background images, SKU color variations and anime storyboards.

The service runs at **`https://mcp.inkroom.ai`** using **Streamable HTTP**. This repository contains the existing standalone Skill client plus public integration examples and documentation. It does not contain the hosted MCP server implementation or a published `@inkroom/mcp-server` npm package.

## Connect to InkRoom

1. Sign in at [InkRoom](https://inkroom.ai/mcp-server), check MCP access and create your personal **MCP Token**. An InkRoom public API Key is a different credential.
2. Add the appropriate configuration below to your client's private MCP settings. Merge the `inkroom` entry with your existing connections; keep other servers and an already-working InkRoom connection.
3. Supply your Token through your client's private configuration or supported environment mechanism. Reconnect and verify tool discovery before generating anything. Installation and tool discovery do not generate images or grant credits.

### Claude Code

Merge into `.mcp.json`. The Token is supplied in the host's `INKROOM_MCP_TOKEN` environment variable, without the `Bearer ` prefix.

```json
{
  "mcpServers": {
    "inkroom": {
      "type": "http",
      "url": "https://mcp.inkroom.ai",
      "headers": {
        "Authorization": "Bearer ${INKROOM_MCP_TOKEN}",
        "Accept-Language": "en",
        "X-InkRoom-Continuation": "receipt-v1"
      }
    }
  }
}
```

Downloadable example: [mcp.claude-code.example.json](mcp.claude-code.example.json). See [Claude Code's MCP documentation](https://code.claude.com/docs/en/mcp) for configuration scopes, variable expansion and private credentials.

### Cursor

Merge into `.cursor/mcp.json` or your private user MCP settings. Cursor's environment expansion uses `${env:NAME}`.

```json
{
  "mcpServers": {
    "inkroom": {
      "url": "https://mcp.inkroom.ai",
      "headers": {
        "Authorization": "Bearer ${env:INKROOM_MCP_TOKEN}",
        "Accept-Language": "en",
        "X-InkRoom-Continuation": "receipt-v1"
      }
    }
  }
}
```

Downloadable example: [mcp.cursor.example.json](mcp.cursor.example.json). The existing [Cursor plugin](cursor/README.md) is another installation option. See [Cursor's MCP documentation](https://cursor.com/docs/mcp) for its current configuration syntax.

Use a host that supports Streamable HTTP and a per-user Authorization header. OAuth-only connectors cannot be assumed compatible with this Bearer-token service. These configurations describe Claude Code and Cursor; they do not establish support in every Claude Desktop, browser-chat or agent product.

Never put a real Token in this public repository, public issues, screenshots or a server URL. Send it only to the configured InkRoom endpoint; do not forward it when downloading original images.

## Tools and execution

Tool visibility depends on Token scopes; models, specifications, prices and account entitlements are checked at use time. Discover the connected tools instead of treating a documentation table as an immutable schema.

| Tool | Purpose |
| --- | --- |
| `help` | Current capability and usage guidance. |
| `list_models` | Available image models, supported parameters, prices and independent processing operations. |
| `generate_image` | Generate an image from a prompt; preview first and continue with its receipt. |
| `derive_image` | Edit from explicit reference images or a resolvable original task. |
| `batch_generate` | Preview and execute a group of images or model comparisons. |
| `upscale_image` | Independent x2/x4 upscaling, anime enhancement or face restoration when available. |
| `wait_images` | Wait for or recover the original task/batch results and their delivery pages. |
| `generation_status` | Read or cancel an existing task, according to the current tool schema. |
| `account`, `spending_policy`, `prompts` | Account/usage, spending authorization and prompt examples; permissions apply. |

Additional tools such as asynchronous batches or delivery-issue reporting may be exposed by your account and current server version. Delivery reports are subject to server verification; a failed local download does not itself establish a refund.

There is no independent `inkroom_remove_background` tool or dedicated mask input in the documented reference-editing workflow. Request background changes through `derive_image` with a reference, then inspect the output. PNG output alone does not prove alpha transparency. Upscaling is separate from general image generation.

1. Discover a supported model and relevant parameters; use the returned pricing rather than hardcoded example prices.
2. Preview the complete request or batch. Show the actual model, parameters and estimated maximum charge.
3. Continue with the returned `requestId` or `batchId` under the account's existing authorization. If the service returns `confirmation_required`, obtain approval for that order before `run`. Never split a batch to evade limits.
4. Keep the original receipt. If waiting, downloading or saving is interrupted, resume it and retrieve the original images instead of generating and charging again.
5. Check dimensions and downloaded files before reporting local delivery complete. Preview images are not original downloads.

Credits are paid funds. Available models and example brand names do not grant access, bypass account limits or guarantee a particular image result.

## Workflows and quality checks

These are creative use cases, not guarantees of exact geometry, colors, compliance or character identity. Use explicit references, compare outputs and review before publication.

| Workflow | Practical input and review | Guide |
| --- | --- | --- |
| GPT Image 2 / Nano Banana / Gemini Image | Query actual available models, allowed specifications and current prices. | [Models and pricing](https://inkroom.ai/en/image/models) |
| Ecommerce image sets | Product reference and shot list; review logos, labels, proportions and material. | [Ecommerce images](https://inkroom.ai/en/solutions/ecommerce-images) |
| White backgrounds | Product reference and white-background request; inspect edge quality, shadows and actual background pixels. | [White-background images](https://inkroom.ai/en/image/product/white-background) |
| SKU recolors | Master product reference and target color; compare texture, highlights and product details. | [Product recolor](https://inkroom.ai/en/image/product/recolor) |
| Handheld product images | Product and scale references; review hand anatomy, grip and product size. | [Handheld images](https://inkroom.ai/en/image/product/handheld) |
| Detail-page imagery | Product reference and layout brief; verify product facts and finish text/layout in your design tools. | [Detail-page images](https://inkroom.ai/en/image/product/detail-page) |
| Anime storyboards | Character references and shot descriptions; compare identity, clothing and continuity across frames. | [Comic and anime images](https://inkroom.ai/en/solutions/comic-anime) |
| Character consistency | Reference sheet and visual anchors; inspect every pose and scene. | [Consistency guide](https://inkroom.ai/en/solutions/character-consistency) |

For example, ask your assistant: “Use InkRoom with this product reference to prepare three listing-image variations. Keep the logo and product proportions. Show the model and estimated charge before execution.” This is a natural-language request, not a ready-made tool payload.

## Agent Skills

A Skill provides workflow instructions; MCP provides the tools. Use the existing [Skill guide](SKILL.md), [Cursor plugin Skill](cursor/skills/inkroom/SKILL.md), or [account installation page](https://inkroom.ai/skill-server). Preserve existing credentials and spending policies. Installation verification should discover tools without generating images.

The existing [standalone client build and recovery instructions](README.md) remain applicable. Remote MCP, the Go client, the website and marketplace packages have separate versions and release states.

## 中文接入指南

InkRoom 面向图片创作者、电商卖家与小型团队，支持文字生图、参考改图、批量、独立放大和原图交付。Remote MCP 地址为 **`https://mcp.inkroom.ai`**，使用 Streamable HTTP；每位用户使用自己的 MCP Token，不与公共 API Key 混用。

本仓库保留现有 Skill 客户端，并提供公开接入配置、使用说明与[市场推广材料](MARKETING.md)。服务端在 InkRoom 托管运行；本文不提供虚构的 npm 安装包，也不把客户端源码称为服务端实现。

Claude Code 使用上面的 `.mcp.json` 配置；Cursor 使用 `.cursor/mcp.json` 配置，两者的环境变量写法不同。将示例中的 `inkroom` 合并到现有配置，保留其他连接；真实 Token 只在客户端的私有设置或安全环境中填写。账户配置入口为 [MCP Server](https://inkroom.ai/mcp-server)。只支持 OAuth 的宿主不能据此视为兼容。

先发现工具与实际可用模型，再预览本次整单费用，按账户已有授权继续；服务端返回 `confirmation_required` 时，先确认本次最高费用再执行。积分按资金处理，不因安装 Skill 或 MCP 自动获得额度。保留 `requestId` / `batchId`；下载或本地保存失败时沿原回执恢复，不重复付费生成。

| 场景 | 入口 |
| --- | --- |
| 模型、规格与当前计费 | [GPT Image 2、Gemini 与图像模型指南](https://inkroom.ai/image/models) |
| 商品系列与批量制图 | [电商图片方案](https://inkroom.ai/solutions/ecommerce-images) |
| 白底、改色、手持与详情图 | [白底图](https://inkroom.ai/image/product/white-background) · [改色](https://inkroom.ai/image/product/recolor) · [手持图](https://inkroom.ai/image/product/handheld) · [详情图](https://inkroom.ai/image/product/detail-page) |
| 漫剧与人物参考 | [漫剧分镜](https://inkroom.ai/solutions/comic-anime) · [角色一致性](https://inkroom.ai/solutions/character-consistency) |
| 海报与商业视觉探索 | [产品海报](https://inkroom.ai/solutions/product-poster) · [商业设计稿](https://inkroom.ai/solutions/design-draft) |
| 安装与方法复用 | [MCP 接入](https://inkroom.ai/solutions/mcp-server) · [Agent Skills](https://inkroom.ai/solutions/agent-skills) |

参考图用于约束结果，不能保证商品、颜色或人物特征绝对不变；逐张核对商标、文字、材质、比例、手部和角色特征，再决定是否发布。MCP 当前没有独立去背景 Tool 或专用蒙版字段；背景修改走 `derive_image`，透明交付须检查原图 alpha。生成图不等同于可编辑分层源文件，也不自动满足电商平台合规要求。

## Public discovery

[Official MCP Registry](https://registry.modelcontextprotocol.io/v0.1/servers/io.github.InkRoomAI%2Finkroom/versions/0.3.23) · [Glama connector](https://glama.ai/mcp/connectors/io.github.InkRoomAI/inkroom) · [Cursor Directory](https://cursor.directory/plugins/inkroom) · [Smithery Skill](https://smithery.ai/skills/inkroom/inkroom)

These links identify different distributions. A listed Skill does not establish a published remote MCP on that platform. Review, publication, actual tool discovery and paid image execution are separate stages.

See [PUBLIC-MATERIALS-LICENSE.md](PUBLIC-MATERIALS-LICENSE.md) for the license scope of these new documents and examples. Existing client, plugin and dependency terms remain in effect.

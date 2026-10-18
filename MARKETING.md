# InkRoom — public messaging and directory submissions

Public material version: **1.0.0** · Reviewed **2026-10-18**

Use these descriptions for InkRoom's existing public client and integration repository. They describe a hosted MCP service with per-user credentials; do not present the repository as the hosted server implementation.

## Brand and links

| Field | Value |
| --- | --- |
| Display name | InkRoom |
| English descriptor | AI image creation, reference editing and batch workflows |
| 中文短介绍 | 面向创作者与电商卖家的 AI 生图、参考改图与批量创作工具。 |
| Public repository | https://github.com/InkRoom-AI/inkroom-client |
| Website | https://inkroom.ai/ |
| English website | https://inkroom.ai/en |
| MCP documentation | https://inkroom.ai/en/solutions/mcp-server |
| Skill documentation | https://inkroom.ai/en/solutions/agent-skills |
| Remote endpoint | https://mcp.inkroom.ai |
| Authentication | Per-user MCP Token in an Authorization: Bearer header |
| Transport | Streamable HTTP |

### GitHub About

Description (under 160 characters):

> Official InkRoom client, MCP integrations and Agent Skills for AI images, reference edits, product visuals, storyboards and batch workflows.

Website: https://inkroom.ai/solutions/mcp-server

Topics:

```text
mcp mcp-server model-context-protocol ai-image-generation ecommerce-photography claude-code cursor-mcp agent-skills character-consistency inkroom
```

The repository contains a client and integration materials. Its description must not imply that the hosted MCP server source is included.

## English descriptions

### Short

InkRoom connects AI assistants to image generation, reference editing, batch workflows and upscaling, with cost previews and original-image delivery.

### Extended

InkRoom helps creators and ecommerce teams produce AI images from prompts and references. Its hosted MCP service supports image generation, reference-based edits, batches, independent upscaling and recovery of original results. Typical workflows include product listing images, white-background variations, SKU recolors and anime storyboards.

Connect a compatible Streamable HTTP client to https://mcp.inkroom.ai with your own InkRoom MCP Token. Query the live model catalog for GPT Image 2, Nano Banana, Gemini Image and other currently available channels; model access, specifications and prices depend on your account and the live service. Preview costs before execution and follow the account's authorization policy.

Keep product and character references explicit, review the generated outputs, and resume original receipts when delivery is interrupted. Agent Skills provide reusable workflow guidance alongside the MCP tools. See [MCP setup](https://inkroom.ai/en/solutions/mcp-server), [Skills](https://inkroom.ai/en/solutions/agent-skills) and [models/pricing](https://inkroom.ai/en/image/models).

### Social/profile copy

Create product visuals and anime images with prompts, references and batches. InkRoom offers Web, API, MCP and Skill workflows, with model and cost previews before execution. https://inkroom.ai/en

### Example calls to action

- Connect your assistant: https://inkroom.ai/en/solutions/mcp-server
- Explore reusable image workflows: https://inkroom.ai/en/solutions/agent-skills
- Check current models and pricing: https://inkroom.ai/en/image/models

## 中文推广文案

### 一句话

InkRoom 把提示词、参考图、AI 生图、改图与批量创作连接起来，帮助创作者和电商团队制作图片、比较结果并复用方法。

### 产品介绍

InkRoom 支持在网页工作台、API、MCP 与 Skill 中进行图片创作。你可以围绕商品实拍图制作白底、改色、手持和详情图，也可以用人物设定图探索漫剧镜头、海报与商业视觉方案。GPT Image 2、Nano Banana、Gemini Image 等渠道的实际可用模型、规格和价格，以账号实时目录及费用预览为准。

在 MCP/Skill 中，先预览整单费用，再按账户已有授权继续；遇到确认要求时先确认本次最高费用。图片生成后逐张检查商品、商标、文字、颜色和人物特征，并保存原图。任务等待或下载中断时沿原回执恢复，避免重复生成和重复扣费。

创作理念：**像搭积木一样创作，像分享链接一样传播。**

### 场景与内容链接

| 场景 | 中文 | English |
| --- | --- | --- |
| 电商系列图 | [电商图片](https://inkroom.ai/solutions/ecommerce-images) | [Ecommerce images](https://inkroom.ai/en/solutions/ecommerce-images) |
| 白底图 | [商品白底图](https://inkroom.ai/image/product/white-background) | [White backgrounds](https://inkroom.ai/en/image/product/white-background) |
| SKU 改色 | [商品改色](https://inkroom.ai/image/product/recolor) | [Product recolor](https://inkroom.ai/en/image/product/recolor) |
| 手持与详情图 | [手持](https://inkroom.ai/image/product/handheld) · [详情](https://inkroom.ai/image/product/detail-page) | [Handheld](https://inkroom.ai/en/image/product/handheld) · [Detail](https://inkroom.ai/en/image/product/detail-page) |
| 漫剧与角色 | [漫剧](https://inkroom.ai/solutions/comic-anime) · [角色一致性](https://inkroom.ai/solutions/character-consistency) | [Storyboards](https://inkroom.ai/en/solutions/comic-anime) · [Characters](https://inkroom.ai/en/solutions/character-consistency) |
| 海报与设计探索 | [海报](https://inkroom.ai/solutions/product-poster) · [设计稿](https://inkroom.ai/solutions/design-draft) | [Posters](https://inkroom.ai/en/solutions/product-poster) · [Design drafts](https://inkroom.ai/en/solutions/design-draft) |

延伸阅读：[电商工具对比方法](https://inkroom.ai/blog/best-ai-ecommerce-photo-tools) · [白底图检查](https://inkroom.ai/blog/amazon-white-background-guide) · [漫剧角色参考与复核](https://inkroom.ai/blog/ai-comic-character-consistency) · [机器可读说明](https://inkroom.ai/llms.txt)

## Directory submission fields

Use the current fields and rules shown by each platform. Existing listings should be updated rather than duplicated. Submission does not establish approval or a public listing.

### mcp.so

- Type: Remote Server, if supported by the current submission form.
- Name: InkRoom.
- Repository: https://github.com/InkRoom-AI/inkroom-client (client and integration documentation).
- Homepage/docs: https://inkroom.ai/en/solutions/mcp-server
- Endpoint: https://mcp.inkroom.ai
- Description: use the short English description above.

At the review date, the standard form advertises a **$39 paid submission**. These materials do not authorize payment. Use a free route only if the platform offers one; otherwise record that a spending decision is required. Do not label a filled form as submitted.

### Smithery

- Name: InkRoom.
- Documentation: https://inkroom.ai/en/solutions/mcp-server
- Endpoint: https://mcp.inkroom.ai
- Authentication: each user's own MCP Token in the Authorization header, including the Bearer prefix.
- Description: use the extended English description above.

The existing [InkRoom Skill](https://smithery.ai/skills/inkroom/inkroom) is separate from remote MCP publication. The earlier remote-MCP onboarding encountered OAuth metadata discovery incompatibility with this Bearer-token service. Recheck the current platform before claiming publication; do not change InkRoom authentication or share a default account Token to pass a scanner. Never put a real Token in a public description or configuration.

### Glama

Existing connector: https://glama.ai/mcp/connectors/io.github.InkRoomAI/inkroom

Use its owner-managed page for maintenance. Preserve the existing connector identity and per-user authentication; do not create a duplicate GitHub-source server entry for this hosted service. Public health/tool discovery is not proof that every paid generation workflow passed acceptance.

### Awesome Remote MCP Servers

The current [awesome-mcp-servers rules](https://github.com/punkpeye/awesome-mcp-servers/blob/main/CONTRIBUTING.md) direct hosted services to [awesome-remote-mcp-servers](https://github.com/punkpeye/awesome-remote-mcp-servers). Follow that repository's current contribution rules, use the existing Glama connector badge, and insert alphabetically in Art & Design:

```markdown
- [InkRoom](https://inkroom.ai/en/solutions/mcp-server) `https://mcp.inkroom.ai`
  [![InkRoom MCP connector](https://glama.ai/mcp/connectors/io.github.InkRoomAI/inkroom/badges/score.svg)](https://glama.ai/mcp/connectors/io.github.InkRoomAI/inkroom)
  🔑 - Generate and edit images from references, run batches, preview costs, upscale and recover originals.
```

PR title: `Add InkRoom remote image creation MCP`

PR body:

> Adds InkRoom to Art & Design with its hosted Streamable HTTP endpoint and existing Glama connector. Users sign up for their own account and supply a personal MCP Token. Tools support image generation, reference edits, batches, cost previews, upscaling and original-result recovery. The public repository contains the client and integration documentation; it is not the hosted server implementation. Maintainers can verify endpoint health through the linked Glama connector.

## Claims and screenshots

Describe references as inputs for consistency; do not promise exact logos, colors, hands, transparency, identity or marketplace approval. Review results before commercial use. Do not invent performance comparisons, directory authority scores, revenue, customers, guaranteed backlinks, search ranking, indexing or AI citation outcomes.

Published visual examples should identify their actual model and relevant inputs when known. Use images the publisher has rights to share, and label mockups as examples. Never include personal Tokens, account balances, private tasks, customer prompts or internal deployment information in screenshots or promotional files.

## Website handoff

Link the public MCP/marketing materials from the existing MCP/Skill landing pages. Keep the brand's verified GitHub organization link and existing marketplace URLs. Software integration links belong with their corresponding software entity; adding a source/documentation link does not prove deployment or search indexing.

Only new public documents and configuration examples named in [PUBLIC-MATERIALS-LICENSE.md](PUBLIC-MATERIALS-LICENSE.md) use that MIT grant. Existing client and plugin terms remain unchanged.

# InkRoom 客户端

## 免 Key 提示词助手（预览）

[InkRoom Prompts MCP 与 Skill](prompts/)：公开提示词和对应示例图，免费限额内文本优化，无需登录或 Key。新增目录采用独立 MIT 许可；线上开通与真实验收待完成。原付费生图授权保持不变。

## MCP、Skill 与公开推广资料

通过 [Remote MCP 接入指南](MCP-GUIDE.md) 配置客户端，查看[创作场景与市场推广材料](MARKETING.md)，或继续使用下方现有独立客户端。品牌 GitHub 账号为 InkRoomAI，公开仓库位于 InkRoom-AI 组织。


[English](README.md) · **[官方网站：inkroom.ai](https://inkroom.ai)**

好模型 高效 批量。通过网站、API、MCP 与 Skill，让 AI 图片创作融入日常工作和自动化流程。

本仓库提供 InkRoom 官方独立客户端源码及 [Skill 使用说明](SKILL.md)，适合电商配图、动漫素材、海报等图片生产场景。图片生成由 InkRoom Remote MCP 服务执行；本客户端不是 MCP 服务端，也不是公共 API SDK。

## 如何选择

| 使用方式 | 适合的工作 |
| --- | --- |
| [网站](https://inkroom.ai) | 日常作图、上传参考图、查看与下载结果 |
| API | 在自己的应用、脚本中集成和批量生产 |
| MCP / Skill | 让 AI 助手调用生图能力，兼顾易用和自动化 |

## 在 WorkBuddy 安装

[在 WorkBuddy 安装「InkRoom 图片创作」](https://www.workbuddy.cn/share/asset?shareCode=OOG16KkXbM3k5ghACZI5x1&sig=L8-dKfDjnKRGnew_)

使用 WorkBuddy 的用户可从此入口安装市场版 Skill，再按包内指引完成账号授权。市场版与本仓库 Go 客户端独立维护；生图需要 InkRoom 账号及有效 MCP Token，安装 Skill 不等于免费生图。

## 安装

**普通用户**：登录 [InkRoom](https://inkroom.ai)，在账号内打开 Skill 配置，使用页面提供的安装与授权指引。以账号页面显示的平台、下载地址和配置为准。

**从源码构建**：需要 Go 1.25.13 或更新版本。

```sh
git clone https://github.com/InkRoom-AI/inkroom-client.git
cd inkroom-client
go mod download
go test ./...
go build -o inkroom .
./inkroom --help
```

Windows 可构建为 `inkroom.exe`，并将下方命令中的 `./inkroom` 替换为 `.\inkroom.exe`。下载的独立可执行文件运行时不需要安装 Go、Python 或 Node.js。

## 最小使用示例

先完成账号页面提供的配置与授权。需要有效 MCP Token；不要把 Token 写进命令参数、示例文件、Git 仓库或截图。客户端支持通过标准输入执行 `authorize`，也支持由自己的安全运行环境提供 `INKROOM_MCP_TOKEN`。

查看当前可用工具及参数：
```sh
./inkroom tools --output ./inkroom-output
```

执行工具调用：
```sh
./inkroom call --tool TOOL_NAME --arguments ./arguments.json --output ./inkroom-output
```

将 `TOOL_NAME` 替换为发现结果中的工具名；`arguments.json` 必须是符合该工具当前 schema 的 JSON 对象。不要将占位命令直接作为生图请求运行，也不要猜测模型 ID、参数或价格。

## 让 AI 助手帮你作图

安装 Skill 后，可以这样描述需求：
- “为这张商品参考图生成 4 张不同构图的配图，保持商品外观一致。先展示实际模型和预计积分。”
- “根据这个海报提示词生成 3 个方案，不要改写我给出的文字。”
- “恢复上一批任务并下载原图；不要重新生成。”

这些是自然语言需求示例，不是固定的工具参数。实际数量、参考图支持、模型及费用，以当前工具和账号权限为准。积分按资金计价。费用预览后遵循账号授权策略；需要额外确认时才执行确认步骤，不能绕过额度或余额限制。

## 下载与恢复

调用结果及媒体清单保存在输出目录。保留原始响应和回执，下载失败时不要重复提交付费生成。

```sh
# 根据原始响应重新尝试保存媒体，不启动新的生图
./inkroom recover --response ./inkroom-output/RESPONSE_FILE.response.json --output ./inkroom-output

# 跟进原任务的只读等待，并保存已完成结果
./inkroom resume --response ./inkroom-output/RESPONSE_FILE.response.json --output ./inkroom-output
```

将响应文件名替换为实际生成的文件。详细的预览、授权、批量、任务恢复和交付规则见 [SKILL.md](SKILL.md)。

## 源码范围与许可

仅包含用户侧客户端和 Skill；不包含服务端源码、账号、上游凭据或业务数据。目前公开源码供查看，**尚未授予独立开源许可证**。第三方依赖保留各自的许可。

[访问 InkRoom](https://inkroom.ai) · [GitHub 组织](https://github.com/InkRoom-AI)



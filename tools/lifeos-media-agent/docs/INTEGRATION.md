# 集成记录

当前仓库是 LifeOS 产品源码。Wails 桌面端以 LifeOS 为产品名，打包本仓库的 Vue 前端，并通过 `apiBase` 转发到 Go API。Go 模块仍沿用 learnos 名称。资料库由专题、分组和 Markdown 页面组成，图片由会话认证的 `/api/v1/library/images` 提供。现有 API Token 允许 `capture:inbox`。媒体接入需要独立权限和契约。桌面端本地配置引用 HTTP，部署 Caddy 使用运行环境的 `APP_DOMAIN`，实际 HTTPS 地址待确认。

当前扫描器仅输出基础元数据。完整字段、封面下载与缓存、SQLite 状态、同步重试队列、远端接口与资料库 UI、Keychain 写入、doctor 和 launchd 安装仍待交付。


## 第二轮验收

- 通过官方文件主机下载 Go 1.26.0 darwin/arm64，解压至临时目录。Agent 的 `go test ./...`、`go vet ./...`、原生构建通过。HTTP 测试需要本地回环监听权限。
- 从本机访问 Calibre OPDS 收到 HTTP 401。已增加 HTTP Basic 和 Keychain 读取入口，认证后的 Feed 尚未验证。
- Jellyfin 公共信息 API 返回 HTTP 200。登录钥匙串未找到 Agent 的 Jellyfin API Key，媒体库读取和封面尚未验证。
- 尚未执行真实同步或部署。同步记录数为 0，远端去重未验证。

## 第四轮本地开发

SQLite 驱动复用 LifeOS 的 go-sqlite3，使用 macOS Command Line Tools 编译。Agent 的写入命令读取 LIFEOS_URL，允许 HTTPS 和显式回环 HTTP。Agent 从 Keychain 服务 lifeos-media-agent-lifeos-token 读取媒体令牌。

通过已登录用户的 POST /api/v1/api-tokens 以 scope=media:sync 创建令牌，沿用原有撤销端点。Agent 向 POST /api/v1/library/media-agent/items 提交媒体元数据。媒体浏览走 GET /api/v1/library/media，前端入口为资料库的媒体中心。Schema 34 增量创建媒体表，元数据冲突更新保留收藏和笔记列。

封面同步、凭证交互设置、完整本地链路验收及 launchd 尚待完成。

## 第五轮本地验收

本地 fixture 提供两页 Calibre Atom Feed 和分页 Jellyfin API。Agent 通过实际 Connector、SQLite 队列和 LifeOS HTTP 接口向隔离数据库写入 5 本书、5 部电影、2 部电视剧。11 张自生成 PNG 通过现有私有图片存储上传并读取，另有一条无封面。首次确认 12 条，重复确认 0 条，改动书名和电影简介后确认 2 条。最终数据库仍为 12 条、11 张图片。

`TestLocalMediaPipeline` 需要 `MEDIA_E2E_BASE_URL` 指向空的隔离 LifeOS 实例，使用测试账号 fixture 和测试密码 fixture-local-only，在回环端口启用实际会话和媒体令牌认证。仅用测试替身提供 NAS 和钥匙串读数。

Chrome 在 1440 和 390 宽度、LifeOS 浅色和深色主题下验收卡片、图片解码、占位、类型及来源筛选、搜索和详情。12 条少于当前每页 24 条，未验收翻页交互。真实 NAS、生产 HTTPS、Keychain 写入和 launchd 均未验收。

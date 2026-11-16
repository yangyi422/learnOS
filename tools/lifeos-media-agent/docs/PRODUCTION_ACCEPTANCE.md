# 生产接入验收

验收环境为 macOS 15.1.1（24B91）、ARM64、Go 1.26.0，系统时区 CST。保留上一轮的 12 条 fixture、截图和验收记录。新增回归使用单独的本地数据库目录。

| 项目 | 实际结果 |
| --- | --- |
| 混合封面 | 本地 HTTP NAS fixture 经 Agent、SQLite、真实 LifeOS API 上传并读回 JPEG、WebP、PNG。12 条媒体包含 11 个封面。 |
| 封面替换 | 仅替换一本书的封面，确认 1 条。该记录的封面 ID 和读回内容 SHA-256 均改变，其余封面 ID 和哈希相同。 |
| 增量与重复 | 首次确认 12 条，无变化确认 0 条，修改书名和电影简介确认 2 条。 |
| 分页 | Chrome 连接独立的 25 条媒体测试库，首页 24 条，尾页 1 条。尾页下一页按钮禁用，返回首页成功。API 第三页为空，第零页返回 HTTP 400。 |
| 登录钥匙串 | 使用独立测试条目验证创建、跨进程读取、更新和清除。真实服务的四个条目均未找到，交互式 secrets set 尚未实测。 |
| Calibre | 实机 HTTP 401，既有观测记录为 Digest MD5 qop=auth。缺少账户和密码，认证后分页、真实元数据、封面和数量待验收。 |
| Jellyfin | 实机公共信息接口 HTTP 200。缺少新 API Key，私有接口、扫描数量和封面待验收。 |
| LifeOS 远端 | Compose 将 APP_DOMAIN 注入 Caddy。当前工作区仅有示例域名，桌面端现有地址为 HTTP，环境中无 APP_DOMAIN 或 LIFEOS_URL。HTTPS 端点、证书、认证、写入和浏览待验收。 |
| 正式 dry-run | doctor 和 sync all --dry-run 均因实际认证或配置缺失退出 1。status 发现正式 state.db 尚未创建。无可信的生产扫描数量、封面数量和写入计划。 |
| 生产数据 | 写入 0 条，封面上传 0 张。未执行迁移、部署或数据库变更。 |
| LaunchAgent | 已实现 service install/status/start/stop/uninstall 和 logs。plist 已通过 plutil，配置为本机时间每日 03:00，最多三次尝试，沿用 Agent 的进程锁。尚未安装、手动触发或验证重启。 |

## 下一步

在登录用户的交互式终端使用正式构建的 Agent 执行：

```sh
media-agent secrets set calibre-username
media-agent secrets set calibre-password
media-agent secrets set jellyfin-api-key
media-agent secrets set lifeos-api-token
```

第四项使用目标 LifeOS 的 media:sync 令牌。将真实 HTTPS 基础地址设为 LIFEOS_URL。重新执行 doctor、sync all --dry-run 和 status。核对目标数据库、最近备份、实际 NAS 数量及首次写入范围，并取得用户明确批准，再执行小批量生产同步。

随后将二进制放在稳定目录，在相同的配置环境中执行 service install。安装会注册每日任务。service start 手动触发，service status 查询，logs 查看最近日志。执行前确保登录钥匙串已解锁。service uninstall 移除任务，保留状态库和日志。

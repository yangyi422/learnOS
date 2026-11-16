# LifeOS Media Agent

当前实现为独立的只读 NAS 扫描器。在 macOS 上运行：

```sh
go run ./cmd/media-agent sync calibre --dry-run
go run ./cmd/media-agent sync jellyfin --dry-run
```

输出为逐行 JSON。Calibre 默认读取任务书中的 OPDS 地址，可用 `CALIBRE_OPDS_URL` 和 `CALIBRE_ALL_BOOKS_HREF` 覆盖。Jellyfin 可用 `JELLYFIN_URL` 覆盖地址，凭证从登录钥匙串中名为 `lifeos-media-agent-jellyfin-api-key` 的通用密码读取。使用重新生成的 Key，可通过钥匙串访问应用录入。

详见 [集成记录](docs/INTEGRATION.md)。

Calibre 若返回 HTTP 401，可设置 `CALIBRE_USERNAME`，并在登录钥匙串中录入服务名为 `lifeos-media-agent-calibre-password` 的通用密码。代理根据服务器的认证挑战选择 HTTP Basic 或 Digest（MD5、qop=auth）。

从本目录执行 `./scripts/go.sh test ./...`、`./scripts/go.sh vet ./...`、`./scripts/go.sh build ./...`。启动脚本支持 `MEDIA_AGENT_GO`、仓库忽略目录 `.tmp/lifeos-toolchain/go/bin/go` 和常用系统安装目录。

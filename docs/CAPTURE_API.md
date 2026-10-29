# 快速采集凭证与 Capture API

面向外部客户端（当前是 `utools-plugin/` 的 uTools 插件）的最小写入通道。设计目标是：第三方客户端能把内容放进 LifeOS 收集箱，但拿不到完整账号权限。

## 为什么不复用会话 Cookie

LifeOS 的登录态是 `HttpOnly` + `SameSite=Lax` 的会话 Cookie，只在浏览器/WebView 同源上下文有效。uTools 插件运行在独立进程的本地页面里，既拿不到这个 Cookie，也不应该拿——那等于把完整账号交给外部程序。因此新增**独立凭证**，与会话互不影响。

## 凭证模型

表 `api_tokens`（schema 33）：

| 列 | 说明 |
| --- | --- |
| `user_id` | 归属用户；凭证只能写自己名下的收集箱 |
| `name` | 显示名称 |
| `token_hash` | `sha256(明文)`，唯一索引；**明文不落库** |
| `prefix` | 前 12 位，仅用于在 UI 中辨认（`los_pat_ab12…`） |
| `scopes` | 逗号分隔，目前只有 `capture:inbox` |
| `expires_at` | 必填；默认 90 天，可设 30/90/180/365 天 |
| `revoked_at` | 撤销时间；非空即失效 |
| `last_used_at` | 最近一次成功认证时间 |

明文格式 `los_pat_<48 hex>`，只在**创建响应**中出现一次，之后无法再次读取。

## 接口

### 管理（仅会话认证，Web 设置页「快速采集凭证」）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/api-tokens` | 列出凭证元数据（不含明文与 hash） |
| `POST` | `/api/v1/api-tokens` | `{ "name": "...", "expires_in_days": 90 }` → `201`，响应含一次性 `token` |
| `DELETE` | `/api/v1/api-tokens/:id` | 撤销 → `200 {"data":{"revoked":true}}` |

`expires_in_days` 允许 1–365；省略或 0 按 90 天。名称超长或有效期越界 → `400 TOKEN_INPUT_INVALID`；撤销不存在的 id → `404 TOKEN_NOT_FOUND`。

### 采集（Bearer token 或会话）

```
POST /api/v1/capture/inbox
Authorization: Bearer los_pat_xxxx
Content-Type: application/json

{ "content": "想法/待办/链接", "capture_key": "cap-<32hex>" }
```

- `201` → `{ "data": { "id", "content", "status", "source_type", "source_url", "created_at" } }`
- 复用 `InboxService.CreateWithKey`：内容 trim、≤10000 字符、URL 自动识别为 `source_type=url` 并记录 `source_url`，记录进入既有收集箱（`inbox_items`），不新建表。
- `capture_key` 可选，格式 8–80 字符 `[A-Za-z0-9_-]`，用于幂等重试：同 key + 同内容返回同一条（`201`），同 key + 不同内容 → `409 CAPTURE_DUPLICATE`。
- 错误：`401 UNAUTHORIZED`（无凭证/凭证无效/已撤销/已过期/用户被停用）、`400 CAPTURE_INPUT_INVALID`、`409 CAPTURE_DUPLICATE`、`5xx CAPTURE_FAILED`。

## 权限边界

- 凭证 principal 的 `Scopes = ["capture:inbox"]`、`Role = user`：即使归属账号是管理员，凭证也**不是**管理员。
- 会话 principal 的 `Scopes` 为 `nil`（完整权限），因此同一路由对 Web 端行为不变。
- `RequireScope` 保护 `/capture/inbox`；其余所有 `/api/v1/*` 仍只接受会话 Cookie，用 Bearer 访问返回 `401`（已有测试覆盖）。
- 撤销、过期、停用用户都会立即失效；`last_used_at` 每次认证更新（失败不影响请求）。

## 安全约束（实现中遵守）

- 明文只出现在创建响应，服务端只存 `sha256`；日志不打印凭证。
- 无「永不过期」选项；上限 365 天。
- 不关闭现有认证、不放宽 CORS、不改 Cookie 策略。因为插件页面与 LifeOS 跨源，插件侧改为**在 preload 的 Node 上下文发起请求**（见 `utools-plugin/preload-request.cjs`），而不是给后端加 CORS。
- 后端不把「未知 token」与「已撤销/已过期」区分返回，统一 `401`。

## 客户端配置（uTools 插件）

1. Web 设置页 →「快速采集凭证」→ 生成 → 复制一次性明文。
2. 打开 uTools，搜索「LifeOS 收集」或「快速记录」。
3. 首次进入的配置区填 API 地址（默认 `http://127.0.0.1:8080`；远程部署填实际地址，如 `http://<host>:8888`）与凭证，保存到 uTools `dbStorage`。
4. 之后每次呼出即聚焦输入，Enter 提交、Shift+Enter 换行。

## 测试

- Go：`internal/service/api_token_service_test.go`（创建/认证/范围/过期/撤销/停用用户/参数校验）、`internal/httpapi/capture_handler_test.go`（token 写入、URL 来源、幂等与 409、缺失/伪造/撤销凭证、空内容与非法 key、生产路由 401）、`internal/database/api_token_migration_test.go`（建表、唯一 hash 索引、重开幂等）。
- 端到端：临时库起真实服务 → 登录 → 发凭证 → Bearer 采集 → 撤销 → 采集 401（`.tmp/e2e-capture.sh`，需本地执行，验证过 `201/401/200/401`）。
- 插件侧：`src/__tests__/preloadRequest.test.ts` 用真实 http 服务验证运输层。

# LearnOS Starter

LearnOS 是一个单用户、自托管的个人 AI 学习系统。本仓库是第一阶段可运行骨架。

## 当前能力

- Go + Gin 后端
- SQLite + GORM 持久化
- Vue 3 + TypeScript + Element Plus 前端
- 自动创建一门营养学示例课程
- `/api/v1/courses` 课程列表接口
- 单用户 Basic Auth
- Docker Compose 与可选 Caddy HTTPS

## 环境要求

- 推荐 Go 1.26+
- 推荐 Node.js 22.18+
- Docker 部署不要求宿主机安装 Go 或 Node

Go 1.26 是当前稳定主版本；Gin 当前文档要求 Go 1.25 或以上。Vue 官方脚手架使用 Vite，当前文档推荐 Node.js 22.18 或更高。

## 本地开发

```bash
cp .env.example .env

# 后端
make dev-api

# 另一个终端启动前端
cd web
npm install
npm run dev
```

访问：`http://localhost:5173`

开发环境中如设置了 Basic Auth，浏览器会弹出账号密码框。

## Docker 运行

```bash
cp .env.example .env
# 部署前修改用户名和密码哈希
docker compose up -d --build app
```

服务仅绑定到宿主机回环地址：`http://127.0.0.1:8080`。

### 生成新密码哈希

本地 Go 环境：

```bash
make hash-password PASSWORD='your-strong-password'
```

也可以使用 Apache `htpasswd`：

```bash
htpasswd -bnBC 12 '' 'your-strong-password' | tr -d ':\n'
```

将结果填入 `.env` 的 `APP_PASSWORD_HASH`。

## 使用域名和 HTTPS

填写 `.env`：

```env
APP_DOMAIN=learn.example.com
```

启动生产 profile：

```bash
docker compose --profile production up -d --build
```

Caddy 将反向代理到应用并自动处理域名证书。

## 验证

```bash
curl http://127.0.0.1:8080/healthz
curl -u admin:change-me http://127.0.0.1:8080/api/v1/courses
```

## 下一步

下一阶段不是继续堆页面，而是实现第一条真实学习闭环：

```text
读取当前知识点
→ 展示问题
→ 提交回答
→ 调用 DeepSeek
→ 保存结构化评价
→ 更新掌握度与误区
```

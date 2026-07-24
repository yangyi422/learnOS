# LearnOS 架构

```text
Browser
  ↓
Vue 3 SPA
  ↓ /api/v1
Go + Gin
  ├── Course Service
  ├── Learning Workflow Service
  ├── Context Builder
  ├── Agent Provider
  ├── Review Scheduler
  └── Markdown Exporter
  ↓
SQLite
  ↓
DeepSeek API
```

## 当前阶段

本骨架只实现：

- Go 服务启动；
- SQLite 初始化与自动迁移；
- 营养学示例课程种子数据；
- 课程列表 API；
- Vue 首页读取并展示课程；
- 单用户 Basic Auth；
- Docker 与 Caddy 部署基础。

## 数据原则

- SQLite 保存原始记录和结构化状态；
- 对话原文用于追溯，不直接作为每次模型上下文；
- 上下文由当前任务、课程状态、相关误区和必要个人档案动态组装；
- Markdown 从数据库生成，默认不与数据库双向编辑。

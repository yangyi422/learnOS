# LearnOS Phase 2：不依赖 AI 的最小学习闭环

请在当前 LearnOS 仓库中实现 **Phase 2：不依赖 AI 的最小学习闭环**。

开始前请先完整阅读：

- `AGENTS.md`
- `README.md`
- `docs/PRODUCT.md`
- `docs/ARCHITECTURE.md`
- `docs/DATABASE.md`
- `docs/ROADMAP.md`
- 现有 Go 后端代码
- 现有 Vue 前端代码
- `Dockerfile`
- `docker-compose.yml`

不要直接重构整个项目。先理解现有代码结构、命名方式、分层方式和 API 风格，再基于现有架构增量实现。

---

## 一、当前阶段目标

实现以下完整链路：

```text
首页点击“继续学习”
→ 进入课程学习页
→ 后端返回当前课程、模块、知识点和问题
→ 用户输入回答
→ 提交回答
→ 后端保存学习记录
→ 返回固定的模拟教学反馈
→ 前端展示反馈
→ 可以查看最近的回答记录
```

本阶段禁止接入 DeepSeek、OpenAI 或其他 LLM。

本阶段重点验证：

- 数据库结构
- 课程状态
- API 设计
- 前后端交互
- 学习记录持久化
- 页面刷新后数据仍然存在

---

## 二、后端技术要求

继续沿用当前项目技术栈：

- Go
- Gin
- GORM
- SQLite
- 现有 Repository / Service / Handler 分层
- 现有错误处理和响应格式

不要新增不必要的第三方依赖。

---

## 三、新增数据库模型

请新增以下模型，并加入 GORM 自动迁移。

### 1. CourseUnit

表示课程模块。

字段建议：

```go
type CourseUnit struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CourseID  uint      `gorm:"not null;index" json:"course_id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Objective string    `gorm:"type:text" json:"objective"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	Status    string    `gorm:"size:32;not null;default:'pending'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

状态暂定：

- `pending`
- `learning`
- `completed`

### 2. Lesson

表示一个可学习的核心知识点。

字段建议：

```go
type Lesson struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	CourseID              uint      `gorm:"not null;index" json:"course_id"`
	UnitID                uint      `gorm:"not null;index" json:"unit_id"`
	Title                 string    `gorm:"size:255;not null" json:"title"`
	CoreQuestion          string    `gorm:"type:text;not null" json:"core_question"`
	ExpectedUnderstanding string    `gorm:"type:text" json:"expected_understanding"`
	SortOrder             int       `gorm:"not null;default:0" json:"sort_order"`
	Status                string    `gorm:"size:32;not null;default:'pending'" json:"status"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
```

状态暂定：

- `pending`
- `learning`
- `completed`

### 3. LearningTurn

保存一次学习问答。

字段建议：

```go
type LearningTurn struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CourseID     uint      `gorm:"not null;index" json:"course_id"`
	UnitID       uint      `gorm:"not null;index" json:"unit_id"`
	LessonID     uint      `gorm:"not null;index" json:"lesson_id"`
	Question     string    `gorm:"type:text;not null" json:"question"`
	UserAnswer   string    `gorm:"type:text;not null" json:"user_answer"`
	Result       string    `gorm:"size:32;not null" json:"result"`
	Feedback     string    `gorm:"type:text" json:"feedback"`
	CorrectParts string    `gorm:"type:text" json:"correct_parts"`
	MissingParts string    `gorm:"type:text" json:"missing_parts"`
	CreatedAt    time.Time `json:"created_at"`
}
```

本阶段 `CorrectParts` 和 `MissingParts` 可以保存 JSON 字符串。

### 4. MasteryRecord

字段建议：

```go
type MasteryRecord struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	CourseID       uint       `gorm:"not null;index" json:"course_id"`
	LessonID       uint       `gorm:"not null;uniqueIndex" json:"lesson_id"`
	MasteryScore   float64    `gorm:"not null;default:0" json:"mastery_score"`
	AnswerCount    int        `gorm:"not null;default:0" json:"answer_count"`
	IncorrectCount int        `gorm:"not null;default:0" json:"incorrect_count"`
	NeedsReview    bool       `gorm:"not null;default:false" json:"needs_review"`
	NextReviewAt   *time.Time `json:"next_review_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
```

### 5. Misconception

字段建议：

```go
type Misconception struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	CourseID              uint      `gorm:"not null;index" json:"course_id"`
	LessonID              uint      `gorm:"not null;index" json:"lesson_id"`
	OriginalUnderstanding string    `gorm:"type:text" json:"original_understanding"`
	CorrectUnderstanding  string    `gorm:"type:text" json:"correct_understanding"`
	BoundaryNotes         string    `gorm:"type:text" json:"boundary_notes"`
	Status                string    `gorm:"size:32;not null;default:'active'" json:"status"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
```

本阶段可以只完成表结构，不要求自动生成 misconception 数据。

---

## 四、初始化示例数据

在现有营养学课程下创建固定数据。

课程：

```text
营养学
```

模块：

```text
水与体液平衡
```

知识点：

```text
标题：口渴是否是可靠的饮水依据

核心问题：
只要不口渴，是否说明身体不缺水？

期望理解：
口渴是身体水分调节的重要信号之一，但不能作为所有场景下唯一的补水依据。
高温、运动、疾病、年龄以及个体感知差异都可能影响口渴信号。
```

要求：

- 初始化逻辑必须幂等
- 重启应用不能重复插入相同模块和知识点
- 可以通过课程名称、模块标题或固定唯一条件判断是否已存在
- 将该知识点设置为当前学习状态
- 将当前模块设置为 `learning`

如果现有 `Course` 表有 `CurrentUnit`、`CurrentUnitID`、`CurrentLessonID` 或类似字段，请复用并正确更新。

如果现有 `Course` 表没有对应字段，请先评估现有设计，在不破坏架构的前提下增加：

```go
CurrentUnitID   *uint
CurrentLessonID *uint
```

不要用字符串保存当前知识点。

---

## 五、后端 API

统一放在现有 `/api/v1` 路由组中。

### 1. 获取当前知识点

```http
GET /api/v1/courses/:id/current-lesson
```

成功响应示例：

```json
{
  "data": {
    "course": {
      "id": 1,
      "name": "营养学"
    },
    "unit": {
      "id": 1,
      "title": "水与体液平衡",
      "objective": ""
    },
    "lesson": {
      "id": 1,
      "title": "口渴是否是可靠的饮水依据",
      "core_question": "只要不口渴，是否说明身体不缺水？",
      "status": "learning"
    }
  }
}
```

要求：

- 校验课程 ID
- 课程不存在返回 404
- 当前知识点不存在时返回明确错误
- 不要把数据库错误原样暴露给前端

### 2. 提交回答

```http
POST /api/v1/courses/:id/answers
```

请求：

```json
{
  "lesson_id": 1,
  "answer": "不正确，口渴只是身体调节水分的一个信号，不能作为唯一依据。"
}
```

校验要求：

- `lesson_id` 必填
- `answer` 去除首尾空格后不能为空
- `answer` 最大长度暂定 5000
- lesson 必须属于当前 course
- lesson 必须是当前课程的当前知识点

本阶段使用固定模拟评价，不调用 AI。

建议固定返回：

```json
{
  "data": {
    "turn_id": 1,
    "result": "mostly_correct",
    "feedback": "你正确认识到口渴不是判断身体水分状态的唯一依据。",
    "correct_parts": [
      "认识到口渴只是身体调节水分的信号之一",
      "认识到不能只依赖单一信号判断"
    ],
    "missing_parts": [
      "还可以补充高温、运动、疾病和年龄等影响因素"
    ],
    "mastery_score": 0.75,
    "needs_review": true
  }
}
```

提交成功后必须：

1. 写入 `learning_turns`
2. 创建或更新 `mastery_records`
3. `AnswerCount + 1`
4. `MasteryScore` 更新为 `0.75`
5. `NeedsReview` 设置为 `true`
6. 暂时不自动切换下一知识点

请使用事务保证 `LearningTurn` 和 `MasteryRecord` 更新的一致性。

### 3. 获取回答历史

```http
GET /api/v1/courses/:id/learning-turns?limit=10
```

要求：

- 默认 `limit=10`
- 最大 `limit=50`
- 按 `created_at` 倒序
- 只返回当前课程记录
- 返回问题、回答、结果、反馈、正确部分、缺失部分、时间

---

## 六、前端页面

继续使用现有 Vue 3、TypeScript、Element Plus。

新增 Vue Router。如果项目已经配置 Router，则复用。

### 路由

```text
/                         学习首页
/courses/:id/learn        课程学习页
/courses/:id/archive      课程档案页
/settings                 系统设置页
```

本阶段重点实现：

```text
/courses/:id/learn
```

### 首页修改

当前首页“继续学习”按钮点击后：

```text
跳转到 /courses/{courseID}/learn
```

课程卡片底部按钮也跳转到对应学习页。

### 学习页布局

页面至少包含：

1. 返回首页按钮
2. 当前课程名称
3. 当前模块名称
4. 当前知识点标题
5. 当前核心问题
6. 多行回答输入框
7. 提交回答按钮
8. 提交中的 loading 状态
9. 提交失败提示
10. 模拟反馈区域
11. 最近学习记录区域

反馈区域分块展示：

- 判断结果
- 回答正确的部分
- 仍然缺少的部分
- 完整反馈
- 掌握度
- 是否需要复习

不要把所有内容堆成一段纯文本。

### 页面交互

进入页面时：

```text
请求 current-lesson
请求 learning-turns
```

提交时：

```text
校验 answer 不为空
→ POST answers
→ 显示反馈
→ 清空或保留回答由你选择，但行为要一致
→ 重新获取 learning-turns
```

刷新页面后历史记录必须仍然存在。

### 前端 API 层

不要在 Vue 组件内散落 `fetch`。

请建立清晰的 API 文件，例如：

```text
web/src/api/courses.ts
web/src/api/learning.ts
```

并定义 TypeScript 类型。

---

## 七、固定反馈的实现方式

固定反馈属于临时模拟实现，请集中放在 Service 层，不要写死在 Handler 里。

建议创建：

```go
type EvaluationResult struct {
	Result       string
	Feedback     string
	CorrectParts []string
	MissingParts []string
	MasteryScore float64
	NeedsReview  bool
}
```

再由 Service 中的临时方法返回：

```go
func mockEvaluateAnswer(answer string) EvaluationResult
```

可以简单根据回答长度做最低限度判断：

- 少于 5 个字符返回 `insufficient`
- 其他情况返回 `mostly_correct`

不要做复杂关键词匹配，因为下一阶段会替换为 DeepSeek。

请在代码中明确标注：

```go
// TODO Phase 3: replace mock evaluation with AI provider.
```

---

## 八、数据库和事务要求

- 所有新增表加入 `AutoMigrate`
- 所有外键 ID 建立索引
- `LearningTurn` 与 `MasteryRecord` 更新使用事务
- Repository 不承担业务判断
- Handler 不直接操作 GORM
- Service 负责课程归属、当前知识点和状态校验
- 数据库错误记录日志，但 API 返回安全错误信息

---

## 九、测试要求

至少增加后端测试，覆盖：

1. 获取存在课程的 current lesson
2. 获取不存在课程返回 404
3. 空回答返回 400
4. 提交有效回答成功
5. `LearningTurn` 被持久化
6. `MasteryRecord` 被创建或更新
7. lesson 不属于 course 时拒绝提交
8. 获取学习历史按时间倒序

如果现有项目暂无测试体系，请使用 Go 标准库 `testing` 和 `httptest`，不要引入重量级测试框架。

前端至少执行：

```bash
npm run build
```

后端至少执行：

```bash
go test ./...
go build ./cmd/server
```

最终还要执行：

```bash
docker compose build app
```

如果当前环境网络原因导致 Docker 构建依赖下载失败，请明确说明，但仍需完成本地静态检查。

---

## 十、文档更新

完成后更新：

- `docs/DATABASE.md`
- `docs/ARCHITECTURE.md`
- `docs/ROADMAP.md`
- `README.md`

在 `ROADMAP` 中把 Phase 2 已完成内容标记出来。

说明当前阶段仍然使用固定模拟评价，DeepSeek 将在 Phase 3 接入。

---

## 十一、不要做的内容

本次不要实现：

- DeepSeek API
- OpenAI API
- Agent 编排
- RAG
- 向量数据库
- 动态课程生成
- 复习调度算法
- 多用户
- 注册登录
- 复杂权限
- WebSocket
- SSE 流式输出
- Markdown 导出
- Obsidian 同步
- 大范围 UI 重构

不要擅自扩大需求范围。

---

## 十二、工作方式

请按照以下步骤执行：

1. 阅读并总结现有项目结构
2. 列出需要新增和修改的文件
3. 给出简短实施计划
4. 实现数据库模型和迁移
5. 实现初始化数据
6. 实现 Repository
7. 实现 Service
8. 实现 Handler 和路由
9. 实现 Vue 路由和学习页面
10. 编写测试
11. 执行格式化、测试、构建
12. 修复发现的问题
13. 更新文档
14. 最后输出：
   - 完成内容
   - 修改文件
   - API 说明
   - 测试结果
   - 尚未完成事项
   - 下一阶段建议

代码要求：

- Go 代码执行 `gofmt`
- TypeScript 不使用 `any`，确有必要时说明原因
- 保持函数职责单一
- 错误信息清晰
- 不在代码中保存密钥
- 不修改 `.env`
- 不删除已有可运行功能
- 不修改 Docker 对外端口
- 不提交数据库文件和运行时 `data` 目录

现在开始先阅读仓库和文档，然后实施 Phase 2。

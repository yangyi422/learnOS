# 数据库演进草案

## 当前已实现

### courses

- id
- name
- description
- goal
- status
- progress
- current_unit
- last_studied_at
- created_at
- updated_at

## 下一阶段计划

- course_units：课程模块
- lessons：最小可提问知识点
- learning_turns：问题、回答和结构化评价
- misconceptions：明确误区及修正状态
- mastery_records：知识点掌握度与复习时间
- practice_tasks：现实实践任务
- agent_runs：模型调用审计与原始输出
- app_settings：DeepSeek 与系统配置

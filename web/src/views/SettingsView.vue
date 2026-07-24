<template>
  <el-card shadow="never">
    <template #header>后续配置项</template>
    <el-descriptions :column="1" border>
      <el-descriptions-item label="模型服务">DeepSeek API</el-descriptions-item>
      <el-descriptions-item label="课程策略">主干 + 个人应用 + 边界探索</el-descriptions-item>
      <el-descriptions-item label="数据存储">SQLite + Markdown 导出</el-descriptions-item>
      <el-descriptions-item label="认证方式">单用户 HTTP Basic Auth</el-descriptions-item>
    </el-descriptions>
  </el-card>
</template>

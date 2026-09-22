<template>
  <section class="page-stack">
    <PageHeader eyebrow="用户管理" title="用户与学习空间" description="管理员可以创建用户、查看用户课程，并暂时停用账号。">
      <template #actions><el-button type="primary" @click="dialogOpen = true">+ 添加用户</el-button></template>
    </PageHeader>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
    <el-table v-loading="loading" :data="users" row-key="id">
      <el-table-column prop="username" label="用户名" />
      <el-table-column prop="display_name" label="显示名称" />
      <el-table-column prop="role" label="角色" />
      <el-table-column label="状态"><template #default="scope"><el-tag :type="scope.row.status === 'active' ? 'success' : 'info'">{{ scope.row.status === 'active' ? '启用' : '停用' }}</el-tag></template></el-table-column>
      <el-table-column label="操作" width="260"><template #default="scope"><el-button link type="primary" @click="showCourses(scope.row)">查看课程</el-button><el-button link @click="toggleStatus(scope.row)">{{ scope.row.status === 'active' ? '停用' : '启用' }}</el-button></template></el-table-column>
    </el-table>

    <el-dialog v-model="dialogOpen" title="添加用户" width="460px">
      <el-form label-position="top">
        <el-form-item label="用户名"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="显示名称"><el-input v-model="form.display_name" /></el-form-item>
        <el-form-item label="初始密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogOpen = false">取消</el-button><el-button type="primary" :loading="saving" @click="save">创建用户</el-button></template>
    </el-dialog>

    <el-dialog v-model="coursesOpen" :title="`${selected?.display_name ?? ''} 的课程`" width="680px">
      <el-table v-loading="coursesLoading" :data="courses"><el-table-column prop="name" label="课程" /><el-table-column prop="status" label="状态" /><el-table-column prop="learning_status" label="学习状态" /><el-table-column prop="mastery_progress" label="掌握度"><template #default="scope">{{ scope.row.mastery_progress }}%</template></el-table-column></el-table>
      <el-empty v-if="!coursesLoading && !courses.length" description="该用户还没有课程" />
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PageHeader from '@/components/PageHeader.vue'
import { createUser, listUserCourses, listUsers, setUserStatus, type CourseSummary, type User } from '@/api/auth'

const users = ref<User[]>([])
const courses = ref<CourseSummary[]>([])
const selected = ref<User>()
const loading = ref(false)
const saving = ref(false)
const coursesLoading = ref(false)
const dialogOpen = ref(false)
const coursesOpen = ref(false)
const error = ref('')
const form = reactive({ username: '', display_name: '', password: '' })

async function load() { loading.value = true; error.value = ''; try { users.value = await listUsers() } catch (reason) { error.value = reason instanceof Error ? reason.message : '用户读取失败' } finally { loading.value = false } }
async function save() { saving.value = true; try { await createUser(form); ElMessage.success('用户已创建'); dialogOpen.value = false; form.username = ''; form.display_name = ''; form.password = ''; await load() } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '用户创建失败') } finally { saving.value = false } }
async function toggleStatus(user: User) { const status = user.status === 'active' ? 'blocked' : 'active'; try { await ElMessageBox.confirm(`确定要${status === 'active' ? '启用' : '停用'} ${user.display_name} 吗？`, '确认操作'); await setUserStatus(user.id, status); await load() } catch (reason) { if (reason !== 'cancel') ElMessage.error(reason instanceof Error ? reason.message : '状态更新失败') } }
async function showCourses(user: User) { selected.value = user; coursesOpen.value = true; coursesLoading.value = true; try { courses.value = await listUserCourses(user.id) } catch (reason) { ElMessage.error(reason instanceof Error ? reason.message : '课程读取失败') } finally { coursesLoading.value = false } }
onMounted(load)
</script>

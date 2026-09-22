<template>
  <article
    class="domain-tile"
    :class="`domain-tile--${identity.tone}`"
    role="button"
    tabindex="0"
    @click="$emit('open')"
    @keydown.enter="$emit('open')"
  >
    <div class="domain-tile__topline">
      <span class="domain-tile__icon" aria-hidden="true">{{ identity.icon }}</span>
      <el-dropdown :disabled="deleting" @command="handleCommand" @click.stop>
        <button class="domain-tile__menu" type="button" :aria-label="`${course.name}课程操作`" :aria-busy="deleting" :disabled="deleting">•••</button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="open">打开学习</el-dropdown-item>
            <el-dropdown-item command="map">查看知识结构</el-dropdown-item>
            <el-dropdown-item command="delete" divided>删除课程</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
    <h3>{{ course.name }}</h3>
    <div class="domain-tile__progress"><span>课程生成度</span><strong>{{ course.generation_progress }}%</strong></div>
    <el-progress :percentage="course.generation_progress" :show-text="false" :stroke-width="4" />
    <div class="domain-tile__progress"><span>学习覆盖度</span><strong>{{ course.coverage_progress }}%</strong></div>
    <el-progress :percentage="course.coverage_progress" :show-text="false" :stroke-width="4" />
    <div class="domain-tile__progress"><span>理解掌握度</span><strong>{{ course.mastery_progress }}%</strong></div>
    <el-progress :percentage="course.mastery_progress" :show-text="false" :stroke-width="4" />
    <p>{{ course.current_unit || '尚未开始' }}</p>
    <StatusTag :status="course.status" />
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deleteCourse } from '@/api/courses'
import StatusTag from '@/components/StatusTag.vue'
import type { Course } from '@/types/course'

const props = defineProps<{ course: Course }>()
const emit = defineEmits<{ open: []; map: []; delete: [] }>()
const deleting = ref(false)

const identities = [
  { tone: 'violet', icon: '◌' },
  { tone: 'green', icon: '⌁' },
  { tone: 'indigo', icon: '✦' },
  { tone: 'orange', icon: '◒' },
  { tone: 'cyan', icon: '⌘' },
]
const identity = computed(() => {
  const key = `${props.course.id}-${props.course.name}`
  const index = [...key].reduce((total, char) => total + char.charCodeAt(0), 0) % identities.length
  return identities[index]
})

async function handleCommand(command: string) {
  if (command === 'open') emit('open')
  if (command === 'map') emit('map')
  if (command !== 'delete' || deleting.value) return

  try {
    await ElMessageBox.confirm(
      `删除“${props.course.name}”后，该课程的结构、学习记录和认知数据将无法恢复。共享知识来源会保留。`,
      '确认删除课程',
      {
        confirmButtonText: '删除课程',
        cancelButtonText: '取消',
        confirmButtonClass: 'el-button--danger',
        type: 'warning',
      },
    )
  } catch {
    return
  }

  deleting.value = true
  try {
    await deleteCourse(props.course.id)
    ElMessage.success(`已删除课程“${props.course.name}”`)
    emit('delete')
  } catch (reason) {
    ElMessage.error(reason instanceof Error ? reason.message : '课程删除失败')
  } finally {
    deleting.value = false
  }
}

</script>

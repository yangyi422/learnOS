<template>
  <el-dialog
    :model-value="open"
    :title="collection ? '编辑专题' : '新建专题'"
    width="560px"
    :before-close="close"
    class="library-dialog"
    @closed="error = ''"
  >
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <el-form label-position="top" @submit.prevent="save"
      ><el-form-item label="专题名称"
        ><el-input
          v-model="form.name"
          aria-label="专题名称"
          maxlength="120" /></el-form-item
      ><el-form-item label="简短说明"
        ><el-input
          v-model="form.description"
          type="textarea"
          :rows="3"
          maxlength="2000"
      /></el-form-item>
      <div class="library-form-grid">
        <el-form-item label="分类（可选）"
          ><el-input
            v-model="form.category"
            placeholder="如：游戏攻略、烹饪"
            maxlength="80" /></el-form-item
        ><el-form-item label="初始模板"
          ><el-select
            v-model="form.template"
            aria-label="专题模板"
            @change="templateChanged"
            ><el-option label="通用资料库" value="general" /><el-option
              label="Wiki 资料库"
              value="wiki" /><el-option
              label="个人菜谱"
              value="recipe" /></el-select></el-form-item
        ><el-form-item label="展示方式"
          ><el-select v-model="form.display_mode"
            ><el-option label="Wiki 目录" value="wiki" /><el-option
              label="卡片清单"
              value="list" /></el-select></el-form-item
        ><el-form-item label="封面 URL（可选）"
          ><el-input v-model="form.cover_url" placeholder="https://…"
        /></el-form-item>
      </div>
      <p class="library-muted">
        模板只影响新页面的初始正文。切换模板或展示方式不会改变已有内容。
      </p></el-form
    >
    <template #footer
      ><el-button :disabled="busy" @click="close()">取消</el-button
      ><el-button type="primary" :loading="busy" @click="save"
        >保存专题</el-button
      ></template
    ></el-dialog
  >
</template>
<script setup lang="ts">
import { reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  saveCollection,
  type Collection,
  type CollectionInput,
} from "@/api/library";
import { localLibraryImage } from "@/utils/libraryMarkdown";
import { safeExternalLink } from "@/utils/externalLinks";
const props = defineProps<{ open: boolean; collection?: Collection }>(),
  emit = defineEmits<{ close: []; saved: [Collection] }>(),
  busy = ref(false),
  error = ref(""),
  form = reactive<CollectionInput>({
    name: "",
    description: "",
    category: "",
    template: "general",
    display_mode: "list",
    cover_url: "",
    version: 0,
  });
let initial = "";
watch(
  () => props.open,
  (value) => {
    if (value) {
      Object.assign(
        form,
        props.collection
          ? {
              name: props.collection.name,
              description: props.collection.description,
              category: props.collection.category,
              template: props.collection.template,
              display_mode: props.collection.display_mode,
              cover_url: props.collection.cover_url,
              version: props.collection.version,
            }
          : {
              name: "",
              description: "",
              category: "",
              template: "general",
              display_mode: "list",
              cover_url: "",
              version: 0,
            },
      );
      initial = JSON.stringify(form);
      error.value = "";
    }
  },
  { immediate: true },
);
function templateChanged() {
  if (!props.collection)
    form.display_mode = form.template === "wiki" ? "wiki" : "list";
}
async function close(done?: () => void) {
  if (busy.value) return;
  if (JSON.stringify(form) !== initial) {
    try {
      await ElMessageBox.confirm(
        "专题信息尚未保存，确定放弃修改？",
        "未保存的修改",
        { confirmButtonText: "放弃修改", cancelButtonText: "继续编辑" },
      );
    } catch {
      return;
    }
  }
  emit("close");
  done?.();
}
async function save() {
  if (busy.value) return;
  if (!form.name.trim()) {
    error.value = "请填写专题名称。";
    return;
  }
  if (
    form.cover_url &&
    !localLibraryImage(form.cover_url) &&
    (!/^https?:\/\//i.test(form.cover_url) || !safeExternalLink(form.cover_url))
  ) {
    error.value = "封面需要有效的 HTTP 或 HTTPS 图片地址。";
    return;
  }
  busy.value = true;
  error.value = "";
  try {
    const row = await saveCollection(props.collection?.id, { ...form });
    initial = JSON.stringify(form);
    ElMessage.success("专题已保存");
    emit("saved", row);
    emit("close");
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
</script>

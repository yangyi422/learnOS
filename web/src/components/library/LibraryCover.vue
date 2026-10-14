<template>
  <div class="library-cover" :class="{ 'is-previewable': preview && !failed }">
    <img v-if="src && !failed" :src="src" :alt="preview ? label : ''"
      loading="lazy" decoding="async" referrerpolicy="no-referrer"
      :tabindex="preview ? 0 : undefined" :role="preview ? 'button' : undefined"
      :aria-label="preview ? `查看大图：${label}` : undefined"
      @error="failed = true" @click="open" @keydown.enter="open"
      @keydown.space.prevent="open" />
    <span v-else class="library-cover-fallback">{{ src ? '封面暂时无法加载' : '暂无封面' }}
      <button v-if="src && preview" type="button" @click.prevent.stop="failed = false">重试</button>
    </span>
    <el-image-viewer v-if="viewing" :url-list="[src]" @close="viewing = false" />
  </div>
</template>
<script setup lang="ts">
import { ref, watch } from 'vue';
const props = defineProps<{ src: string; label: string; preview?: boolean }>();
const failed = ref(false), viewing = ref(false);
watch(() => props.src, () => { failed.value = false; viewing.value = false; });
function open() { if (props.preview && props.src && !failed.value) viewing.value = true; }
</script>

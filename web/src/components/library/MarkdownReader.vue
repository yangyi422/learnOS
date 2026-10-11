<template>
  <div class="library-markdown" @click="follow" v-html="html" />
</template>
<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from "vue";
import { useRouter, useRoute } from "vue-router";
import { checkPageLinks } from "@/api/library";
import { markdownHTML, internalPageID } from "@/utils/libraryMarkdown";
const props = defineProps<{
    body: string;
    collection: number;
    revision?: number;
  }>(),
  router = useRouter(),
  route = useRoute(),
  available = ref(new Map<number, boolean>());
const html = computed(() =>
  markdownHTML(props.body, props.collection, available.value),
);
let sequence = 0;
onBeforeUnmount(() => sequence++);
watch(
  () => [props.body, props.collection, props.revision],
  async () => {
    const seq = ++sequence;
    available.value = new Map();
    const ids = [
      ...new Set(
        Array.from(
          props.body.matchAll(/\/explore\/library\/\d+\/pages\/\d+/g),
          (m) => internalPageID(m[0], props.collection),
        ).filter((id): id is number => id !== null),
      ),
    ];
    for (let i = 0; i < ids.length; i += 100) {
      try {
        const states = await checkPageLinks(
          props.collection,
          ids.slice(i, i + 100),
        );
        if (seq !== sequence) return;
        const updated = new Map(available.value);
        Object.entries(states).forEach(([id, value]) =>
          updated.set(Number(id), value),
        );
        available.value = updated;
      } catch {
        /* keep unavailable checks pending; reread retries without false deletion claims */
      }
      if (seq !== sequence) return;
    }
  },
  { immediate: true },
);
function follow(event: MouseEvent) {
  const a = (event.target as HTMLElement).closest("a[data-library-page]");
  if (!a) return;
  event.preventDefault();
  const id = Number(a.getAttribute("data-library-page"));
  if (available.value.get(id))
    router.push({
      path: `/explore/library/${props.collection}/pages/${id}`,
      query: route.query,
    });
}
</script>

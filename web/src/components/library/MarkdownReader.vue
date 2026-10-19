<template>
  <div
    class="library-markdown"
    @click="follow"
    @error.capture="imageFailed"
    @keydown="imageKey"
    v-html="html"
  />
  <el-image-viewer
    v-if="previewImage"
    :url-list="[previewImage]"
    @close="previewImage = ''"
  />
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
    articleTitle?: string;
  }>(),
  router = useRouter(),
  route = useRoute(),
  available = ref(new Map<number, boolean>());
const previewImage = ref("");
const html = computed(() => {
  const container = document.createElement('div');
  container.innerHTML = markdownHTML(props.body, props.collection, available.value);
  // Keep the stored Markdown intact; the application owns the page's sole h1.
  container.querySelectorAll('h1').forEach(heading => {
    if (heading === container.firstElementChild && heading.textContent?.trim() === props.articleTitle?.trim()) {
      heading.remove();
    } else {
      const section = document.createElement('h2');
      section.append(...heading.childNodes);
      heading.replaceWith(section);
    }
  });
  return container.innerHTML;
});
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
function imageFailed(event: Event) {
  const img = event.target;
  if (!(img instanceof HTMLImageElement) || img.hidden) return;
  img.hidden = true;
  const span = document.createElement("span");
  span.className = "library-image-failed";
  span.setAttribute("role", "status");
  span.append(
    document.createTextNode(
      `${img.alt || "图片"}：无法加载，请确认登录或图片是否存在。 `,
    ),
  );
  const retry = document.createElement("button");
  retry.type = "button";
  retry.textContent = "重试";
  retry.setAttribute("data-image-retry", "");
  span.append(retry);
  img.after(span);
}
function imageKey(event: KeyboardEvent) {
  if (
    event.target instanceof HTMLImageElement &&
    (event.key === "Enter" || event.key === " ")
  ) {
    event.preventDefault();
    previewImage.value = event.target.src;
  }
}
function follow(event: MouseEvent) {
  const target = event.target as HTMLElement;
  const retry = target.closest("button[data-image-retry]");
  if (retry) {
    const span = retry.parentElement;
    const img = span?.previousElementSibling;
    if (img instanceof HTMLImageElement) {
      img.hidden = false;
      img.src = img.getAttribute("src") || "";
      span?.remove();
    }
    return;
  }
  if (target instanceof HTMLImageElement) {
    event.preventDefault();
    previewImage.value = target.src;
    return;
  }

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

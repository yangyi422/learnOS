<template><div class="review-records"><p v-if="!items.length" class="life-muted">{{ empty }}</p><article v-for="item in items" :key="item.key" class="review-record"><div class="life-metadata"><time>{{ item.date }}</time><span>{{ item.state }}</span><span v-if="item.milestone">◇ 里程碑</span><span v-if="item.domain">{{ domainLabel(item.domain) }}</span></div><h3>{{ item.title }}</h3><p v-if="item.detail" class="review-record-detail">{{ item.detail }}</p><RouterLink v-if="item.available" :to="item.url">查看原始记录</RouterLink><span v-else class="life-muted">原始来源已不可用，保留当时摘要。</span><RouterLink v-if="item.related_event_id && item.related_available" :to="{path:'/life',query:{event:item.related_event_id}}" class="review-related">查看生活收录</RouterLink><span v-else-if="item.related_event_id" class="life-muted">生活收录来源已不可用。</span></article></div></template>
<script setup lang="ts">
import type { ReviewItem } from '@/api/reviews'
import { domainLabel } from '@/api/life'
defineProps<{ items: ReviewItem[]; empty: string }>()
</script>

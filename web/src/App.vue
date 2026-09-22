<script setup lang="ts">
import AppShell from '@/components/AppShell.vue'
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <RouterView v-if="route.path === '/login'" />
  <AppShell v-else />
</template>

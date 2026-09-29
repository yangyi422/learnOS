<template>
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
    <template v-if="name === 'home'"><path d="m3 10 9-7 9 7v10a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1z"/><path d="M9 21v-7h6v7"/></template>
    <template v-else-if="name === 'inbox'"><path d="M4 4h16l1 11-3 5H6l-3-5z"/><path d="M3.5 14h5l1.5 2h5l1.5-2h5"/></template>
    <template v-else-if="name === 'courses'"><rect x="3" y="4" width="7" height="16" rx="1.5"/><rect x="10" y="4" width="7" height="16" rx="1.5"/><path d="m18 7 3 12M5.5 8h2M12.5 8h2"/></template>
    <template v-else-if="name === 'projects'"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 10h18M9 10v10M15 10v10"/></template>
    <template v-else-if="name === 'explore'"><circle cx="12" cy="12" r="9"/><path d="m15.5 8.5-2.2 4.8-4.8 2.2 2.2-4.8z"/></template>
    <template v-else-if="name === 'settings'"><path d="M4 7h16M4 17h16"/><circle cx="9" cy="7" r="2" fill="var(--bg-surface)"/><circle cx="16" cy="17" r="2" fill="var(--bg-surface)"/></template>
    <template v-else-if="name === 'users'"><circle cx="9" cy="8" r="3"/><path d="M3.5 20v-2a5.5 5.5 0 0 1 11 0v2zM16.5 5.5a3 3 0 0 1 0 5.5M17 14a5 5 0 0 1 3.5 5"/></template>
    <template v-else-if="name === 'logout'"><path d="M10 4H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h5M14 7l5 5-5 5M8 12h11"/></template>
    <template v-else-if="name === 'menu'"><path d="M4 7h16M4 12h16M4 17h16"/></template>
    <template v-else-if="name === 'sun'"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M2 12h2m16 0h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></template>
    <template v-else-if="name === 'moon'"><path d="M20 15.5A8.5 8.5 0 0 1 8.5 4 8.5 8.5 0 1 0 20 15.5z"/></template>
    <template v-else-if="name === 'system'"><rect x="3" y="4" width="18" height="14" rx="2"/><path d="M8 21h8M12 18v3"/></template>
    <template v-else-if="name === 'drag'"><circle cx="8" cy="5" r="1"/><circle cx="16" cy="5" r="1"/><circle cx="8" cy="12" r="1"/><circle cx="16" cy="12" r="1"/><circle cx="8" cy="19" r="1"/><circle cx="16" cy="19" r="1"/></template>
  </svg>
</template>

<script setup lang="ts">
defineProps<{ name: 'home' | 'inbox' | 'courses' | 'projects' | 'explore' | 'settings' | 'users' | 'logout' | 'menu' | 'sun' | 'moon' | 'system' | 'drag' }>()
</script>

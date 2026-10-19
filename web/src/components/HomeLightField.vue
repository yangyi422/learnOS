<template>
  <div class="home-light-field" aria-hidden="true" />
</template>

<style scoped>
/* Broad light enters from outside the workspace; the content never sits on a light core. */
.home-light-field { position: absolute; inset: 0; pointer-events: none; }
.home-light-field::before { position: absolute; inset: 0; content: ''; background-image:
  radial-gradient(ellipse 76% 48% at -12% 24%,var(--home-light-entry) 0%,transparent 82%),
  radial-gradient(ellipse 65% 52% at 112% 66%,var(--home-light-reflection) 0%,transparent 85%),
  radial-gradient(ellipse 70% 30% at 12% 115%,var(--home-light-floor) 0%,transparent 88%);
}
</style>

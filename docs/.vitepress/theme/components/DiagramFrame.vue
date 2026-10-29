<script setup>
import { useData, withBase } from "vitepress";
import { computed, onMounted, ref } from "vue";

const props = defineProps({
  src: { type: String, required: true },
  title: { type: String, required: true },
  height: { type: Number, default: 640 },
});

const { isDark, lang } = useData();

// Vue hydration leaves SSR attribute values in place, so the theme query is
// only added on the client; otherwise a dark-mode visitor would get the
// server's light diagram until the next appearance toggle.
const mounted = ref(false);
onMounted(() => {
  mounted.value = true;
});

const themeQuery = computed(() =>
  mounted.value ? `theme=${isDark.value ? "dark" : "light"}` : "",
);
const frameSrc = computed(() =>
  mounted.value
    ? `${withBase(props.src)}?embed=1&${themeQuery.value}`
    : undefined,
);
const openSrc = computed(
  () => withBase(props.src) + (themeQuery.value ? `?${themeQuery.value}` : ""),
);
const openLabel = computed(() =>
  lang.value.startsWith("zh") ? "全屏打开" : "Open full screen",
);
</script>

<template>
  <figure class="kite-diagram-frame">
    <iframe
      :src="frameSrc"
      :title="title"
      :style="{ height: `${height}px` }"
      loading="lazy"
    />
    <figcaption>
      <span>{{ title }}</span>
      <a :href="openSrc" target="_blank" rel="noopener">{{ openLabel }} ↗</a>
    </figcaption>
  </figure>
</template>

<style scoped>
.kite-diagram-frame {
  margin: 16px 0 24px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  overflow: hidden;
  background: var(--vp-c-bg-soft);
}

.kite-diagram-frame iframe {
  display: block;
  width: 100%;
  border: 0;
  background: var(--vp-c-bg-soft);
}

.kite-diagram-frame figcaption {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-top: 1px solid var(--vp-c-divider);
  color: var(--vp-c-text-2);
  font-size: 12px;
  line-height: 20px;
}

.kite-diagram-frame figcaption a {
  flex-shrink: 0;
  color: var(--vp-c-brand-1);
  text-decoration: none;
}

.kite-diagram-frame figcaption a:hover {
  text-decoration: underline;
}
</style>

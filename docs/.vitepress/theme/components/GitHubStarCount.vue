<script setup>
import { onMounted, ref } from "vue";

defineProps({
  ariaLabel: {
    type: String,
    default: "View KubeOsprey stars on GitHub",
  },
});

const starCount = ref(0);

onMounted(async () => {
  try {
    const response = await fetch("https://api.github.com/repos/fadewalk/kube-osprey");

    if (response.ok) {
      const repository = await response.json();

      if (typeof repository.stargazers_count === "number") {
        starCount.value = repository.stargazers_count;
      }
    }
  } catch {}
});
</script>

<template>
  <a
    class="star-count"
    href="https://github.com/fadewalk/kube-osprey/stargazers"
    target="_blank"
    rel="noreferrer"
    :aria-label="ariaLabel"
  >
    <span aria-hidden="true">★</span>
    {{ starCount.toLocaleString("en-US") }}
  </a>
</template>

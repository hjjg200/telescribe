<template>
  <nav class="sidebar"
    :class="{wide: wide}">
    <button class="wide-toggle"
      @click="wide = !wide">
      <FontAwesome v-if="wide" icon="arrow-left"/>
      <FontAwesome v-else icon="arrow-right"/>
    </button>
    <slot/>
  </nav>
</template>

<script>
import { library } from '@fortawesome/fontawesome-svg-core';
import { faArrowRight, faArrowLeft } from '@fortawesome/free-solid-svg-icons';
library.add(faArrowRight, faArrowLeft);
export default {
  name: "Sidebar",
  data() {
    return {
      wide: false,
    };
  }
}
</script>
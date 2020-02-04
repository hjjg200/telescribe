<template>
  <div class="dropdown"
    @click="open = !open"
    v-click-outside="function() {open = false;}">
    <div class="title">{{ name }}</div>
    <div class="items" v-show="open">
      <slot/>
    </div>
  </div>
</template>

<script>
import vClickOutside from 'v-click-outside';
import { library } from '@fortawesome/fontawesome-svg-core';
import { faCaretDown } from '@fortawesome/free-solid-svg-icons';
library.add(faCaretDown);

export default {
  name: "Dropdown",
  props: ["name"],
  directives: { clickOutside: vClickOutside.directive },
  data() {
    return {
      open: false
    };
  }
}
</script>
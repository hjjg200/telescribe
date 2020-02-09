<template>
  <div class="dropdown"
    :class="{open: open}"
    @click="open = !open"
    v-click-outside="function() {open = false;}">
    <div class="title">
      {{ selected ? selected.text : "..." }}
    </div>
    <div class="caret">
      <font-awesome icon="caret-down"/>
    </div>
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
  directives: { clickOutside: vClickOutside.directive },
  data() {
    return {
      open: false,
      selected: undefined,
      items: []
    };
  },
  methods: {
    selectItem(item) {
      this.selected = item;
    }
  }
}
</script>
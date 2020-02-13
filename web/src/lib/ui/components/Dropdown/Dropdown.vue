<template>
  <div class="dropdown"
    :class="{open: open}"
    @click="open = !open"
    v-click-outside="function() {open = false;}">
    <div class="button">
      <span v-if="title !== ''" class="title">{{ title }}</span>
      <span class="value">{{ selected ? selected.text : "..." }}</span>
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
      items: [],
      title: ""
    };
  },
  methods: {
    selectItem(item) {
      this.selected = item;
      this.$emit("change", item.value);
    },
    selectIndex(i) {
      this.selectItem(this.items[i]);
    }
  }
}
</script>
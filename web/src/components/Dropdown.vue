<template>
  <div class="dropdown"
    :class="{ open: open }"
    v-click-outside="function() { open = false; }">

    <div class="title-wrap">
      <div class="category-wrap"
        @click="open = true">
        <div class="category">{{ category }}</div>
        <FontAwesome class="caret" icon="caret-down"/>
      </div>
    </div>

    <div class="menu-wrap" v-if="open">
      <div class="category-wrap"
        @click="open = false">
        <div class="category">{{ category }}</div>
        <FontAwesome class="caret" icon="caret-up"/>
      </div>
      <ul class="items-wrap">
        <li class="item"
          :class="{ selected: valueOf(selected) === valueOf(item) }"
          v-for="item in items"
          :key="valueOf(item)"
          :value="valueOf(item)"
          @click="select(item)">
          {{ labelOf(item) }}
        </li>
      </ul>
    </div>

  </div>
</template>

<script>
import { library } from '@fortawesome/fontawesome-svg-core';
import { faCaretUp, faCaretDown } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
library.add(faCaretUp, faCaretDown);

import vClickOutside from 'v-click-outside';

export default {
  name: "Dropdown",
  components: { 'FontAwesome': FontAwesomeIcon },
  directives: { clickOutside: vClickOutside.directive },
  props: ["category", "items", "default"],
  data() {
    return {
      open: false,
      selected: undefined
    };
  },
  mounted() {
    this.select(this.default);
  },
  methods: {
    valueOf(item) {
      if(item === undefined) return undefined;
      return item.value || item;
    },
    labelOf(item) {
      if(item === undefined) return undefined;
      return item.label || this.valueOf(item);
    },
    select(item) {
      this.open = false;
      this.selected = item;
      this.$emit('select', this.valueOf(item));
    }
  }
}
</script>

<style lang="scss" scoped>
$pad-l: .75em;
ul {
  list-style: none;
  padding: 0;
}
.dropdown {
  position: relative;
}
.title-wrap, .menu-wrap {
  border: 1px solid #eee;
  border-radius: .5em;
  background: white;
  overflow: hidden;
}
.menu-wrap {
  position: absolute;
  top: 0;
  left: 0;
  z-index: 10000;
}
.category-wrap {
  position: relative;
}
.category, .item {
  line-height: inherit;
  padding-left: $pad-l;
  padding-right: 3.5em;
  user-select: none;
  -webkit-user-select: none;
}
.category {
  font-weight: bold;
}
.item:hover {
  background: rgba(0, 0, 0, 0.02);
}
.item.selected {
  color: red;
}
.caret {
  color: #eee;
  width: .8rem;
  height: .8rem;
  top: calc(50% - .4rem);
  right: $pad-l;
  position: absolute;
}
</style>
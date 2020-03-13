<template>
  <div class="ui-dropdown"
    :class="{open: open}"
    v-click-outside="function() {open = false;}">
    <div class="button"
      v-show="hasButton"
      @click="open = !open">
      <Button>{{ title }}</Button>
      <div class="caret">
        <font-awesome icon="caret-down"/>
      </div>
    </div>
    <div class="items"
      v-show="open"
      @click="open = multiple ? true : false">
      <slot/>
    </div>
  </div>
</template>

<script>
import Button from '../Button/Button.vue';
import vClickOutside from 'v-click-outside';
import { library } from '@fortawesome/fontawesome-svg-core';
import { faCaretDown } from '@fortawesome/free-solid-svg-icons';
library.add(faCaretDown);

export default {
  name: "Dropdown",
  components: {Button},
  directives: { clickOutside: vClickOutside.directive },
  props: {
    title: {
      type: String, default: "Dropdown"
    },
    hasButton: {
      type: Boolean, default: true
    }
  },
  data() {
    return {
      open: false
    };
  },
  methods: {
  }
}
</script>
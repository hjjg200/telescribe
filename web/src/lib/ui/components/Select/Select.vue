<template>
  <div class="ui-select"
    :class="{open: open}"
    v-click-outside="function() {open = false;}">
    <div class="button"
      v-show="hasButton"
      @click="open = !open">
      <div class="display">{{ selected ? selected : "..." }}</div>
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
import vClickOutside from 'v-click-outside';
import { library } from '@fortawesome/fontawesome-svg-core';
import { faCaretDown } from '@fortawesome/free-solid-svg-icons';
library.add(faCaretDown);

export default {
  name: "Select",
  directives: { clickOutside: vClickOutside.directive },
  props: {
    name: {
      type: String, default: ""
    },
    hasButton: {
      type: Boolean, default: true
    },
    multiple: Boolean,
    selected: {} // v-model
  },
  model: {
    prop: "selected",
    event: "change"
  },
  data() {
    return {
      open: false
    };
  },
  methods: {
    hasValue(value) {
      if(this.selected)
        return this.multiple
          ? this.selected.indexOf(value) !== -1
          : this.selected === value;
      return false;
    },
    selectValue(ddit) {

      let newVal;
      let value = ddit.value;
      let html  = ddit.$el.htmlContent;
      if(this.multiple) {
        if(this.selected) {
          let copy = this.selected.slice(0);
          const i  = copy.indexOf(value);
          if(i > -1) copy.splice(i, 1);
          else       copy.push(value);
          newVal = copy;
        } else {
          newVal = [value];
        }
      } else {
        if(this.selected !== value) {
          newVal = value;
        }
      }
      this.$emit("change", newVal);

    }
  }
}
</script>
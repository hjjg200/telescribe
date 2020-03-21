<template>
  <div class="ui-select"
    :class="{
      'has-selected': selected.length > 0
    }"
    v-click-outside="function() {open = false;}">

    <div class="button"
      @click="open = !open">

      <!-- form support -->
      <select v-model="selectedValue" :name="name" :multiple="multiple"
        size="1" ref="select" @focus="focused = true">
        <option v-for="item in items" :key="item.value" :value="item.value">{{ item.text }}</option>
      </select>

      <div class="title">{{ text }}</div>
      <div class="caret">
        <font-awesome icon="caret-down"/>
      </div>
      
    </div>
    <div class="items"
      v-show="open"
      v-always-in-viewport="flexible"
      @click="open = multiple ? true : false">
      <slot/>
    </div>
  </div>
</template>

<script>
import mixinFocusable from '../../mixins/focusable.js';
import mixinToggledClass from '../../mixins/toggledClass.js';
import {library} from '@fortawesome/fontawesome-svg-core';
import {faCaretDown} from '@fortawesome/free-solid-svg-icons';
library.add(faCaretDown);

export default {
  name: "Select",
  mixins: [
    mixinFocusable,
    mixinToggledClass('open')
  ],
  props: {
    name: {
      type: String, default: ""
    },
    title: {
      type: String, default: "Select"
    },
    multiple: Boolean,
    selected: {},
    selectedValue: {} // v-model
  },
  model: {
    prop: "selectedValue",
    event: "change"
  },
  data() {
    return {
      items: []
    };
  },
  computed: {
    text() {
      let items  = this.selected;
      let length = items.length;
      if(length === 0) return this.title;

      let text = items[0].text;
      if(this.multiple && length > 1) {
        return `${text} + ${length - 1}`;
      }
      return text;
    },
    selected() {
      return this.items.filter(
        d => this.hasSelected(d)
      );
    }
  },
  methods: {
    hasSelected(item) {
      let value = item.value;
      if(this.selectedValue)
        return this.multiple
          ? this.selectedValue.indexOf(value) !== -1
          : this.selectedValue === value;
      return false;
    },
    selectItem(item) {

      let newVal;
      if(this.multiple) {
        if(this.selected) {
          let copy = this.selected.slice(0);
          const i  = copy.indexOf(item);
          if(i > -1) copy.splice(i, 1);
          else       copy.push(item);
          newVal = copy.map(d => d.value);
        } else {
          newVal = [item.value];
        }
      } else {
        if(this.selected !== item) {
          newVal = item.value;
        }
      }
      this.$emit("change", newVal);

    }
  }
}
</script>
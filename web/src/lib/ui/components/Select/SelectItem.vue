<template>
  <div class="ui-select-item"
    :class="{'is-selected': selected}"
    @click.stop="onClick">
    <div v-show="$parent.multiple" class="checkbox">
      <Checkbox readonly="true" v-model="selected"/>
    </div>
    <div class="text"><slot/></div>
  </div>
</template>

<script>

import {getChildrenTextContent} from '../../util/util.js';
export default {
  name: "SelectItem",
  props: {
    value: {}
  },
  created() {
    this.parentSelect.items.push(this);
  },
  computed: {
    selected() {
      return this.parentSelect.hasSelected(this);
    },
    text() {
      return getChildrenTextContent(this.$slots.default);
    },
    parentSelect() {
      let p;
      for(let x = 0; x <= 5; x++) {
        p = this.$parent;
        let name = p.$options.name;
        let i    = ["Select", "SelectGroup"].indexOf(name);
        if(i === -1)     break;
        else if(i === 0) return p;
      }
      return undefined;
    }
  },
  methods: {
    onClick(event) {
      this.parentSelect.selectItem(this);
      this.$emit("click", event);
    }
  }
}
</script>
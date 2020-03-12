<template>
  <label class="ui-checkbox"
    :class="{checked: checked}"
    @change.stop="onChange">
    <input type="checkbox" :value="value" :readonly="readonly" :checked="checked">
    <div class="mark">
      <font-awesome v-show="checked" icon="check"/>
    </div>
    <div class="label">
      <slot/>
    </div>
  </label>
</template>

<script>
import { library } from '@fortawesome/fontawesome-svg-core';
import { faCheck } from '@fortawesome/free-solid-svg-icons';
library.add(faCheck);
export default {
  name: "Checkbox",
  props: {
    name: {
      type: String, default: ""
    },
    value: {},
    model: {},
    readonly: {
      type: Boolean, default: false
    }
  },
  model: {
    prop: "model",
    event: "change"
  },
  computed: {
    checked() {
      return this.isGroup
        ? this.model.indexOf(this.value) !== -1
        : this.model;
    },
    isGroup() {
      return Array.isArray(this.model);
    }
  },
  methods: {
    onChange(ev) {
      let tf = ev.target.checked;
      let copy;
      if(this.isGroup) {
        copy = [].concat(this.model);
        tf
          ? copy.push(this.value)
          : copy.splice(copy.indexOf(this.value), 1);
      } else {
        copy = tf;
      }
      this.isChecked = tf;
      this.$emit("change", copy);
    }
  }
}
</script>
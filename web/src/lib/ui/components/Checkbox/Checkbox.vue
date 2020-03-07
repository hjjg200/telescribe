<template>
  <label class="checkbox"
    :class="{checked: checked}"
    @change.stop="onChange">
    <input type="checkbox"
      :value="value">
    <div class="mark"
      :class="markClass"
      :style="{'background-color': color}">
      <font-awesome v-if="checked" icon="check"/>
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
      type: String,
      default: ""
    },
    value: {
      type: String
    },
    "_model": {
      // v-model
    },
    markClass: {
      type: Array,
      default: []
    },
    color: {
      type: String,
      default: ""
    }
  },
  model: {
    prop: "_model",
    event: "change"
  },
  computed: {
    checked() {
      return this.isGroup
        ? this._model.indexOf(this.value) !== -1
        : this._model;
    },
    isGroup() {
      return Array.isArray(this._model);
    }
  },
  methods: {
    onChange(ev) {
      let tf = ev.target.checked;
      let copy;
      if(this.isGroup) {
        copy = [].concat(this._model);
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
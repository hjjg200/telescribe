<template>
  <label class="checkbox"
    :class="{checked: checked}">
    <input type="checkbox"
      :value="value"
      v-model="checked">
    <div class="mark">
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
    checked: {
      type: Boolean,
      default: false
    }
  }
}
</script>
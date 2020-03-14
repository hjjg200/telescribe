<template>
  <span class="ui-text-icon"
    :style="{'background-color': colorify(text)}"
    ><span class="text">{{ text.substring(0, 2) }}</span>
  </span>
</template>

<script>
import {colorify} from '../../util/util.js';
export default {
  name: "TextIcon",
  props: {
    text: String
  },
  methods: {
    colorify(str) {
      return colorify(str);
    }
  }
}
</script>
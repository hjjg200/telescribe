<template>
  <div class="ui-rule"
    :class="{
      horizontal: hTypes.indexOf(type) !== -1,
      vertical: vTypes.indexOf(type) !== -1
    }"></div>
</template>

<script>

let hTypes = ["h", "hr", "horizontal"];
let vTypes = ["v", "vr", "vertical"];
import mixinVariant from '../../mixins/variant.js';
export default {
  name: "Rule",
  mixins: [mixinVariant],
  data() {
    return {
      hTypes, vTypes
    };
  },
  props: {
    type: String // v, vr, vertical, h, hr, horizontal
  }
}
</script>
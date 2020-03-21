<template>
  <button class="ui-button"
    @click="$emit('click', $event)">
    <slot/>
  </button>
</template>

<script>
import mixinVariant from '../../mixins/variant.js';
import mixinToggledClass from '../../mixins/toggledClass.js';
export default {
  name: "Button",
  mixins: [
    mixinVariant,
    mixinToggledClass('disabled')
  ],
  props: {
    type: String
  }
}
</script>
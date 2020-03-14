<template>
  <button class="ui-button"
    :class="{
      'type-accent': type === 'accent'
    }"
    @click="$emit('click')">
    <slot/>
  </button>
</template>

<script>
export default {
  name: "Button",
  props: {
    type: String
  }
}
</script>
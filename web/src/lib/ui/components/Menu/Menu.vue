<template>
  <div class="ui-menu"
    :class="{open: open}"
    v-show="open"
    v-always-in-viewport="debug">
    <slot/>
  </div>
</template>

<script>
export default {
  name: "Menu",
  data() {
    return {
      open: true
    };
  }
}
</script>
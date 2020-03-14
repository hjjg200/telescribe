<template>
  <div class="ui-menu"
    :class="{open: open}"
    v-show="open">
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
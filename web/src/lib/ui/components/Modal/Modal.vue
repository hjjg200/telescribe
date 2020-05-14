<template>
  <div class="ui-modal"
    :class="{open: open}"
    v-show="open">
    <div class="ui-modal-window">
      <div class="ui-modal-close" @click="open = false">&times;</div>
      <div class="ui-modal-content">
        <slot/>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "Modal",
  data() {
    return {
      open: false
    }
  }
}
</script>
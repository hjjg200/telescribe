<template>
  <div class="item">
    <slot @click="change"/>
  </div>
</template>

<script>
export default {
  name: "DropdownItem",
  props: [
    "value"
  ],
  methods: {
    change() {
      this.$emit('change', this.value);
    }
  }
}
</script>
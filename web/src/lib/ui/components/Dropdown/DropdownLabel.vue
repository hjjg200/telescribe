<template>
  <label style="display: none"><slot/></label>
</template>

<script>
export default {
  name: "DropdownLabel",
  mounted() {
    if(this.$parent.$options.name === "Dropdown") {
      this.$parent.title = this.$el.textContent;
      this.remove();
    }
  }
}
</script>
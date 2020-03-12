<template>
  <div class="ui-dropdown-item" @click="onClick">
    <div class="checkbox">
      <Checkbox readonly="true" :checked="selected" />
    </div>
    <div class="item">
      <slot/>
    </div>
  </div>
</template>

<script>
export default {
  name: "DropdownItem",
  props: {
    value: {}
  },
  computed: {
    selected() {
      return this.$parent.hasValue(this.value);
    }
  },
  methods: {
    onClick() {
      this.$parent.selectValue(this.value);
      this.$emit("click");
    }
  }
}
</script>
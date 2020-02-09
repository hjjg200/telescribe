<template>
  <div class="item" @click="onClick">
    <slot/>
  </div>
</template>

<script>
export default {
  name: "DropdownItem",
  props: {
    value: {},
    selected: {
      type: Boolean,
      default: false
    }
  },
  created() {
    this.$parent.items.push(this);
  },
  mounted() {
    this.text = this.$el.textContent;
    if(this.selected) {
      this.$parent.selectItem(this);
    }
  },
  methods: {
    onClick() {
      this.$parent.selectItem(this);
    }
  }
}
</script>
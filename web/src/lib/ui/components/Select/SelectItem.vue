<template>
  <div class="ui-select-item" @click="onClick">
    <div v-show="$parent.multiple" class="checkbox">
      <Checkbox readonly="true" v-model="selected"/>
    </div>
    <div class="item">
      <slot/>
    </div>
  </div>
</template>

<script>
export default {
  name: "SelectItem",
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
      this.$parent.selectValue(this);
      this.$emit("click");
    }
  }
}
</script>
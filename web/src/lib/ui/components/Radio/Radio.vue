<template>
  <label class="ui-radio"
    :class="{checked: checked}"
    @change.stop="onChange">
    <input ref="input" :name="name" type="radio" :value="value" :readonly="readonly">
    <div class="mark">
      <div class="circle"></div>
    </div>
    <div class="label">
      <slot/>
    </div>
  </label>
</template>

<script>
export default {
  name: "Radio",
  props: {
    name: {
      type: String, default: ""
    },
    value: {},
    model: {
      default: undefined
    },
    readonly: {
      type: Boolean, default: false
    }
  },
  model: {
    prop: "model",
    event: "change"
  },
  computed: {
    checked() {
      return this.model === this.value;
    }
  },
  methods: {
    onChange(ev) {
      let tf = ev.target.checked;
      if(tf) this.$emit("change", this.value);
    }
  }
}
</script>
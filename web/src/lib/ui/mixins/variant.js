

export default {
  props: {
    variant: {
      type: String, default: ""
    }
  },
  mounted() {
    if(this.variant !== "") {
      this.$el.classList.add(`v-${this.variant}`);
    }
  }
}
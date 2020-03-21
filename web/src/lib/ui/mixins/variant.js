

export default {
  props: {
    variant: {
      type: String, default: ""
    }
  },
  mounted() {
    this.updateVariant();
  },
  watch: {
    variant() {
      this.updateVariant();
    }
  },
  methods: {
    updateVariant() {
      if(this.variantClassName)
        this.$el.classList.remove(this.variantClassName);
      
      if(this.variant !== "") {
        this.variantClassName = `v-${this.variant}`;
        this.$el.classList.add(this.variantClassName);
      } else {
        this.variantClassName = undefined;
      }
    }
  }
}
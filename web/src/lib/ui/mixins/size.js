
let sSizes = ["s", "sm", "small"];
let mSizes = ["m", "md", "medium"];
let lSizes = ["l", "lg", "large"];
let sizes = {
  small:  sSizes,
  medium: mSizes,
  large:  lSizes
};
export default {
  props: {
    size: String
    // s, sm, small
    // m, md, medium
    // l, lg, large
  },
  mounted() {
    this.updateSize();
  },
  watch: {
    size() {
      this.updateSize();
    }
  },
  methods: {
    updateSize() {
      if(this.sizeClassName) {
        this.$el.classList.remove(this.sizeClassName);
      }

      if(this.size !== "") {
        for(let size in sizes) {
          if(sizes[size].indexOf(this.size) !== -1) {
            this.sizeClassName = `sz-${size}`;
            this.$el.classList.add(this.sizeClassName);
            return;
          }
        }
      }

      this.sizeClassName = undefined;
    }
  }
}
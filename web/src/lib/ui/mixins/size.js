
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
    for(let size in sizes) {
      if(sizes[size].indexOf(this.size) !== -1) {
        this.$el.classList.add(`sz-${size}`);
        return;
      }
    }
  }
}
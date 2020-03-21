
import mixinToggledClass from './toggledClass.js';

export default {
  mixins: [
    mixinToggledClass('focused')
  ],
  mounted() {
    let el = this.$el;
    el.tabIndex = 0;
  },
  events: {
    focus(event) {
      this.focused = true;
    },
    blur(event) {
      this.focused = false;
    }
  }
}
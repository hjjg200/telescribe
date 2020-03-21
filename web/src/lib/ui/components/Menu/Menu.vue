<template>
  <div class="ui-menu"
    :class="{open: open}"
    v-show="open"
    v-always-in-viewport="flexible"
    v-click-outside="onClickOutside"
    @click="open = false">
    <slot/>
  </div>
</template>

<script>
export default {
  name: "Menu",
  data() {
    return {
      open: false,
      togglers: []
    };
  },
  methods: {
    toggle(event) {
      let t = event.target;
      if(this.togglers.indexOf(t) === -1) this.togglers.push(t);
      this.open = !this.open;
    },
    onClickOutside(event) {
      let t = event.target;
      if(this.togglers.indexOf(t) === -1) this.open = false;
    }
  }
}
</script>
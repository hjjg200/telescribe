<template>
  <div class="sidebar-item">
    <div class="icon">
      <span v-if="icon === ''">{{ iconFallback }}</span>
    </div>
    <div class="details">
      <slot/>
    </div>
  </div>
</template>

<script>
export default {
  name: "SidebarItem",
  props: {
    icon: {
      type: String,
      default: ""
    }
  },
  data() {
    return {
      iconFallback: ""
    };
  },
  mounted() {
    let titleEl = this.$el.querySelectorAll("strong, b")[0];
    this.iconFallback = titleEl.textContent.substring(0, 2);

    if(this.icon !== '') {
      this.$el.style.backgroundImage = `url("${this.icon}")`;
    }
  }
}
</script>
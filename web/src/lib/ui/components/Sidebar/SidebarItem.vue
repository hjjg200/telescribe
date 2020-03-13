<template>
  <div class="ui-sidebar-item"
    @click="$emit('click')">
    <div class="icon">
      <span v-if="icon === ''">{{ iconFallback }}</span>
    </div>
    <div class="details">
      <slot/>
    </div>
  </div>
</template>

<script>
import {colorify} from '@/lib/ui/util/util.js';

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
    let icon  = this.$el.querySelector(".icon");
    let title = this.$el.querySelectorAll("strong, b")[0].textContent;
    this.iconFallback = title.substring(0, 2);
    icon.style.backgroundColor = colorify(title);

    if(this.icon !== '') {
      icon.style.backgroundImage = `url("${this.icon}")`;
    }
  }
}
</script>
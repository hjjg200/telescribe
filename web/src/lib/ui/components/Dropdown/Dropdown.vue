<template>
  <div class="dropdown" @click="open = !open">
    <div class="title">{{ name }}</div>
    <div class="items" v-show="open">
      <slot/>
    </div>
  </div>
</template>

<script>
export default {
  name: "Dropdown",
  props: ["name"],
  data() {
    return {
      open: false
    };
  }
}
</script>
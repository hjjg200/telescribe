<template>
  <div class="ui-menu-item" @click="$emit('click', $event)">
    <slot/>
  </div>
</template>

<script>
export default {
  name: "MenuItem"
}
</script>
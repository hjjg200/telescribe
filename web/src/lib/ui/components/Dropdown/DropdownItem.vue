<template>
  <div class="ui-dropdown-item"
    @click="$emit('click')">
    <slot/>
  </div>
</template>

<script>
export default {
  name: "DropdownItem"
}
</script>
<template>
  <span class="ui-badge">
    <slot/>
  </span>
</template>

<script>
import mixinSize from '../../mixins/size';
import mixinVariant from '../../mixins/variant';
export default {
  name: "Badge",
  mixins: [mixinSize, mixinVariant]
}
</script>
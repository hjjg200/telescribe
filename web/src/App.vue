<template>
  <div id="app">
    <ul class="client-list">
      <Client v-for="(client, fullName) in abstract.clientMap"
        :key="fullName"
        :abstract="client"/>
    </ul>
  </div>
</template>

<script>
import Client from './components/Client.vue';
export default {
  name: "App",
  components: { Client },
  data() {
    return {
      abstract: this.$root.abstract,
      options: this.$root.options
    };
  }
}
</script>

<style lang="scss">
@import url(//fonts.googleapis.com/css?family=Open+Sans:400,800&display=swap);
</style>
<style lang="scss" src="./style.scss"></style>
<template>
  <div id="app">
    <Sidebar/>
    <div class="client-container">
      <ul class="client-list">
        <Client v-for="(client, fullName) in abstract.clientMap"
          :key="fullName"
          :abstract="client"/>
      </ul>
    </div>
  </div>
</template>

<script>
import Client from './components/Client.vue';
import Sidebar from './components/Sidebar.vue';
export default {
  name: "App",
  components: {
    Client, Sidebar
  },
  data() {
    return {
      clientMap: {},
      abstract: this.$root.abstract,
      options: this.$root.options
    };
  }
}
</script>

<style lang="scss">
@import url(//fonts.googleapis.com/css?family=Open+Sans:400,700&display=swap);
</style>
<style lang="scss" src="./style.scss"></style>
<template>
  <div id="app">
    <main>
      <Sidebar>
        <SidebarLabel>Hosts</SidebarLabel>
        <SidebarItem v-for="(client, fullName) in abstract.clientMap"
          :key="fullName"
          @click="visibleClient = fullName">
          <strong>{{ fullName }}</strong>
          Details
        </SidebarItem>
        <SidebarLabel class="version">{{ version }}</SidebarLabel>
      </Sidebar>
      <section>
        
        <Client v-for="(client, fullName) in abstract.clientMap"
          :key="fullName"
          :body="client"
          :class="{visible: (visibleClient === fullName)}"></Client>
      </section>
    </main>
  </div>
</template>

<script>
import Client from '@/components/Client.vue';
export default {
  name: "App",
  components: {Client},
  computed: {
    abstract() {
      return this.$root.abstract;
    },
    options() {
      return this.$root.options;
    },
    version() {
      return this.$root.version;
    }
  },
  data() {
    return {
      visibleClient: undefined
    };
  }
}
</script>

<style lang="scss">
@import url(//fonts.googleapis.com/css?family=Open+Sans:400,700&display=swap);
</style>
<style lang="scss" src="@/style.scss"></style>
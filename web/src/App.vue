<template>
  <div id="app">
    <main>
      <Sidebar>
        <SidebarLabel>Hosts</SidebarLabel>
        <SidebarItem v-for="(clInfo, clId) in clMap"
          :key="clId"
          @click="visibleClient = clId">
          <strong>{{ clInfo.alias }}</strong>
          clInfo.host
        </SidebarItem>
        <SidebarLabel class="version">{{ version }}</SidebarLabel>
      </Sidebar>
      <section>
        
        <Client v-for="(clInfo, clId) in clMap"
          :key="clId"
          :info="clInfo"
          :class="{visible: (visibleClient === clId)}"></Client>
      </section>
    </main>
  </div>
</template>

<script>
import Client from '@/components/Client.vue';
export default {
  name: "App",
  components: {Client},
  data() {
    let {clMap, webCfg, version} = this.$root;
    return {
      clMap, webCfg, version,
      visibleClient: undefined
    };
  }
}
</script>

<style lang="scss">
@import url(//fonts.googleapis.com/css?family=Open+Sans:400,700&display=swap);
</style>
<style lang="scss" src="@/style.scss"></style>
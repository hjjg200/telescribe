<template>
  <div id="app">
    <header>

      <div class="header-logo">
        Telescribe
      </div>
      <div class="menu-container">

        <Button class="menu-button" @click="$refs.menu.toggle($event)">
          <font-awesome icon="bars"/>
        </Button>
        <Menu ref="menu" class="menu-content">
          <MenuLabel>Favorites</MenuLabel>
          <MenuLabel>Clients</MenuLabel>
          <MenuItem
            v-for="(info, clId) in clMap"
            :key="clId"
            @click="visibleClient = clId">
            <div class="client-menu-item">
              <div class="thumbnail">
                <TextIcon :text="clId"/>
              </div>
              <Icon :type="statusIconOf(clStatMap[clId])" />
              <div class="text">
                <div class="alias">{{ info.alias }}</div>
                <div class="host">{{ info.host }}</div>
              </div>
            </div>
          </MenuItem>
        </Menu>

      </div>
    </header>
    <hr class="dark"/>
    <main>
      <section>


        <Client v-for="(clInfo, clId) in clMap"
          :key="clId"
          :info="clInfo"
          :statusMap="clStatMap[clId]"
          :class="{visible: (visibleClient === clId)}"></Client>
      </section>
    </main>
  </div>
</template>

<script>
import {statusIconOf} from '@/lib/util/web.js';
import {library} from '@fortawesome/fontawesome-svg-core';
import {faBars} from '@fortawesome/free-solid-svg-icons';
library.add(faBars);
import Client from '@/components/Client.vue';
export default {
  name: "App",
  components: {Client},
  data() {
    let {clMap, clStatMap, webCfg, version} = this.$root;
    return {
      clMap, clStatMap, webCfg, version,
      visibleClient: undefined
    };
  },
  methods: {
    statusIconOf
  }
}
</script>

<style lang="scss">
@import url(//fonts.googleapis.com/css?family=Open+Sans:400,700&display=swap);
</style>
<style lang="scss" src="@/style.scss"></style>
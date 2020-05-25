<template>
  <div id="app">
    <header>

      <div class="header-logo">
        <Logo/>
      </div>
      <div class="menu-container">

        <Button class="menu-button" @click="$refs.menu.toggle($event)">
          <font-awesome icon="bars"/>
        </Button>

        <Menu ref="menu" class="menu-content">
          <MenuLabel>Favorites</MenuLabel>
          <MenuLabel>Clients</MenuLabel>
          <MenuItem
            v-for="(clientInfo, clientId) in clientInfoMap"
            :key="clientId"
            @click="visibleClient = clientId">
            <div class="client-menu-item">
              <div class="thumbnail">
                <TextIcon :text="clientId"/>
              </div>
              <Icon :type="statusIconOf(itemStatusMap[clientId], true)" />
              <div class="text">
                <div class="alias">{{ clientInfo.alias }}</div>
                <div class="host">{{ clientInfo.host }}</div>
              </div>
            </div>
          </MenuItem>
        </Menu>

      </div>
    </header>
    <Rule type="hr"/>

    <main>
      <section>

        <Cover v-show="visibleClient === null">
          Select a Client
        </Cover>
        <Client v-for="(clientInfo, clientId) in clientInfoMap"
          :key="clientId"
          :info="clientInfo"
          :itemStatusMap="itemStatusMap[clientId]"
          :class="{visible: (visibleClient === clientId)}"></Client>

      </section>
    </main>
    <footer>

      <Rule type="hr"/>
      <section>
        <div class="version">{{ version }}</div>
        <div class="github">
          <a href="https://github.com/hjjg200/telescribe">https://github.com/hjjg200/telescribe</a>
        </div>
      </section>

    </footer>
  </div>
</template>

<script>
import {statusIconOf} from '@/lib/util/web.js';
import {library} from '@fortawesome/fontawesome-svg-core';
import {faBars} from '@fortawesome/free-solid-svg-icons';
library.add(faBars);
import Logo from '@/components/Logo.vue';
import Client from '@/components/Client.vue';
export default {
  name: "App",
  components: {Client, Logo},
  async created() {

    let {clientInfoMap} = await this.$api.v1.getClientInfoMap();
    let itemStatusMap = {};
    for(let clientId in clientInfoMap) {
      try {
        itemStatusMap[clientId] = (await this.$api.v1.getClientItemStatus(clientId)).clientItemStatus;
      } catch(ex) {
        continue;
      }
    }

    this.itemStatusMap = itemStatusMap;
    this.clientInfoMap = clientInfoMap;

  },
  data() {
    return {
      clientInfoMap: {},
      itemStatusMap: {},
      visibleClient: null
    };
  },
  methods: {
    statusIconOf
  }
}
</script>

<style lang="scss">
@import url(//fonts.googleapis.com/css?family=Lato:300,400,900&display=swap);
</style>
<style lang="scss" src="@/style.scss"></style>
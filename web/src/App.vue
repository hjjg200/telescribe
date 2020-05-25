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
          
          <template v-for="tp in [['Favorites', true], ['Clients', false]]">

            <MenuLabel>{{ tp[0] }}</MenuLabel>
            <MenuItem
              v-for="(clientInfo, clientId) in clientInfoMap"
              :key="clientId"
              v-if="isFavorite(clientId) === tp[1]"
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
            
          </template>

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

    // Check for storage
    if(typeof(Storage) !== "undefined") {
      // Favorites are stored in JSON array form
      let favorites = localStorage.getItem("telescribe.favorites") || "[]";
      favorites = JSON.parse(favorites);
      this.favorites = favorites;
    }

  },
  data() {
    return {
      clientInfoMap: {},
      itemStatusMap: {},
      favorites:     [],
      visibleClient: null
    };
  },
  methods: {
    statusIconOf,
    isFavorite(clientId) {
      return this.favorites.indexOf(clientId) !== -1;
    },
    toggleFavorite(clientId) {
      let i = this.favorites.indexOf(clientId);
      i !== -1 ? this.favorites.splice(i, 1) : this.favorites.push(clientId);

      if(typeof(Storage) !== "undefined") {
        localStorage.setItem("telescribe.favorites", JSON.stringify(this.favorites));
      }
    }
  }
}
</script>

<style lang="scss">
@import url(//fonts.googleapis.com/css?family=Lato:300,400,900&display=swap);
</style>
<style lang="scss" src="@/style.scss"></style>
<template>
  <div class="sidebar-wrap"
    :class="{ full: full }">
  
    <div class="toggle"
      @click="full = !full">
      <FontAwesome v-if="full" icon="arrow-left"/>
      <FontAwesome v-else icon="arrow-right"/>
    </div>

    <div class="sidebar">

      <h4>Clients</h4>
      <ul class="client-list">
        <li v-for="(client, fullName) in app.clientMap"
          class="client"
          :key="fullName"
          :data-fullName="fullName"
          :data-status="client.status()"
          @click="select(fullName)">
          <span class="icon-wrap">
            <span class="icon">{{ fullName.substr(0, 2) }}</span>
            <span class="status"></span>
          </span>
          <span class="name">{{ fullName }}</span>
        </li>
      </ul>

    </div>

  </div>
</template>

<script>
export default {
  name: "Sidebar",
  data() {
    let app = this.$parent;
    return {
      app: app,
      full: false,
      liMap: {}
    };
  },
  updated() {
    // li
    var lis = this.$el.querySelectorAll(".client");
    for(let i = 0; i < lis.length; i++) {
      let li = lis[i];
      this.liMap[li.getAttribute("data-fullName")] = li;
    }
  },
  methods: {
    select(fullName) {
      var map = this.app.clientMap;
      for(let key in map) {
        let el = map[key].$el;
        let li = this.liMap[key];
        let action = key === fullName ? "add" : "remove";

        // Action
        el.classList[action]("visible");
        li.classList[action]("selected");
      }
    }
  }
};
</script>
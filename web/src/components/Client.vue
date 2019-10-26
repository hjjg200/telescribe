<template>
  <li class="client">
    <div class="client-header">
      <span class="name">{{ fullName }}</span>
    </div>
    <div class="client-body">
      <ul class="md-list">
        <li v-for="(latest, key) in latestMap" :key="key" :data-status="status(key)">
          <div class="label-wrap">
            <label>
              <input type="checkbox"
                @change="update()"
                :value="key" v-model="activeKeys">
              <div class="key-value-wrap">
                <span class="key">{{ key }}</span>
                <span class="value">{{ latest.value }}</span>
              </div>
            </label>
          </div>
        </li>
      </ul>
      <div class="chart-options">
        <Dropdown category="Duration"
          :items="durations"
          :default="durations[0]"
          @select="chart.duration($event)"/>
      </div>
      <div class="chart-wrap">
        <Chart :abstract="abstract"/>
      </div>
    </div>
  </li>
</template>

<script>
import Chart from '@/components/Client/Chart.vue';
import Dropdown from '@/components/Dropdown.vue';
export default {
  name: "Client",
  components: { Chart, Dropdown },
  props: ['abstract'],
  data() {
    return {
      app: this.$parent,
      activeKeys: [],
      keys: Object.keys(this.abstract.latestMap),
      fullName: this.$vnode.key,
      csvBox: this.abstract.csvBox,
      latestMap: this.abstract.latestMap,
      configMap: this.abstract.configMap
    };
  },
  computed: {
    durations() {
      var $ = this;
      return this.app.options.durations.map(function(i) {
        return {label: $._shortDuration(i), value: i}
      });
    }
  },
  created: function() {
    // Due to the limitations of modern JavaScript (and the abandonment of Object.observe),
    // Vue cannot detect property addition or deletion.
    // Vue.set or Vue.prototype.$set is required
    this.$set(this.app.clientMap, this.fullName, this);
  },
  mounted: function() {
    var $ = this;
    this.checkboxes = function() {
      var obj = {};
      $.keys.forEach(function(key) {
        obj[key] = $.$el.querySelector(`input[type="checkbox"][value="${key.escapeQuote()}"]`);
      });
      return obj;
    }();
  },
  methods: {

    _shortDuration: function(t) {
      if(t <= 3600) return Math.round(t / 60) + "m";
      else if(t <= 24 * 3600) return Math.round(t / 3600) + "h";
      else return Math.round(t / 86400) + "d";
    },

    status: function(key) {
      let map = this.latestMap;
      if(key === undefined) {
        let max = -1;
        for(let key in map) {
          let st = map[key].status;
          max = st > max ? st : max;
        }
        return max;
      }
      return map[key].status;
    },

    update: function() {
      for(let key in this.checkboxes) {
        let box = this.checkboxes[key];
        var i = this.activeKeys.indexOf(box.value);
        if(i === -1) box.className = "";
        else box.className = i.toSeries();
      }
      this.chart.keys(this.activeKeys);
    }

  }
}
</script>
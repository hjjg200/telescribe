<template>
  <li class="client">
    <div class="client-header">
      <span class="status" :data-status="status()"></span>
      <span>{{ fullName }}</span>
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
      <div class="timeframe">
        <button v-for="duration in app.options.durations"
          :key="duration" 
          @click="chart.duration(duration)">{{ duration }}</button>
      </div>
      <div class="chart-wrap">
        <Chart :abstract="abstract"/>
      </div>
    </div>
  </li>
</template>

<script>
import Chart from './Client/Chart.vue';
export default {
  name: "Client",
  components: { Chart },
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
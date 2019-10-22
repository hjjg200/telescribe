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
  props: ['key', 'abstract'],
  data() {
    return {
      app: this.$parent,
      fullName: this.key
    };
  }
}
</script>
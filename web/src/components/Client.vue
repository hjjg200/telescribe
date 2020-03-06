<template>
  <article class="client">
    <h2>{{ info.alias }}</h2>
    <ul class="checkboxes">
      <li v-for="(latest, mKey) in info.latestMap"
        :key="mKey" :data-status="status(mKey)">
        <Checkbox :value="mKey" v-model="activeKeys"
          :class="[classLegend(mKey), 'legend']">
          <span class="key">{{ mKey }}</span>
          <span class="value">{{ latest.value.format() }}</span>
        </Checkbox>
      </li>
    </ul>
    <div class="options-wrap">
      <Dropdown ref="durations" name="duration" @change="onDurationChange">
        <DropdownLabel>Duration</DropdownLabel>
        <DropdownItem v-for="(duration, i) in $root.webCfg.durations"
          :key="i" :value="duration"
          :selected="i == 0">{{ formatDuration(duration) }}</DropdownItem>
      </Dropdown>
    </div>
    <div class="graph-wrap">
      <Graph ref="graph"/>
    </div>
  </article>
</template>

<script>

import Queue from '@/lib/util/queue.js';
export default {
  name: "Client",
  created() {
    let $ = this;

    // Clean up first
    $.dataset = {};
    let boundaries = [];

    this.queue.queue(new Promise(resolve => {
      $.$api.v1.getMonitorDataBoundaries($.id)
        .then(function(csv) {
          $.$d3.csvParse(csv, row => {
            boundaries.push(+row.timestamp);
          });
          $.$refs.graph.boundaries = boundaries;
          resolve();
        });
    }));
  },
  mounted() {
    let $ = this;
    this.queue.queue(new Promise(resolve => {
      $.$refs.durations.selectIndex(0);
      resolve();
    }));
  },

  props: {
    info: {
      type: Object
    }
  },
  data() {
    return {
      activeKeys: [],
      dataset: {},
      queue: new Queue()
    };
  },
  computed: {
    id() {
      return this.$vnode.key;
    }
  },
  watch: {
    activeKeys(newVal, oldVal) {
      var $ = this;
      var activeDataset = {};
      var ensure = (i = 0) => {
        if(i >= newVal.length) {
          $.$refs.graph.dataset = activeDataset;
          return;
        }

        var mKey = newVal[i];
        if($.dataset[mKey] == undefined) {
          (async function() {
            var p = new Promise(resolve => {
              // Make buf so as not to invoke watchers
              let buf = [];

              $.$api.v1.getMonitorDataTable($.id, mKey)
                .then(function(csv) {
                  $.$d3.csvParse(csv, row => {
                    buf.push({
                      timestamp: +row.timestamp,
                      value: +row.value
                    });
                    $.$set($.dataset, mKey, buf);
                    resolve();
                  });
                });
            });
            await p;
            activeDataset[mKey] = $.dataset[mKey];
            ensure(++i);
          })();
        } else {
          activeDataset[mKey] = $.dataset[mKey];
          ensure(++i);
        }
      };

      ensure();
    }
  },
  methods: {
    classLegend(mKey) {
      var i = this.activeKeys.indexOf(mKey);
      return i.toSeries();
    },
    status(mKey) {
      let map = this.info.latestMap;
      if(mKey === undefined) {
        let max = -1;
        for(let mKey in map) {
          let st = map[mKey].status;
          max = st > max ? st : max;
        }
        return max;
      }
      return map[mKey].status;
    },
    formatDuration(t) {
      if(t <= 60) return `${t}m`;
      else if(t <= 24 * 60) return Math.round(t / 60) + "h";
      else return Math.round(t / 1440) + "d";
    },
    onDurationChange(val) {
      this.$refs.graph.duration = val;
    }
  }
}
</script>
<template>
  <article class="client">
    <h2>{{ fullName }}</h2>
    <ul class="checkboxes">
      <li v-for="(latest, key) in body.latestMap"
        :key="key" :data-status="status(key)">
        <Checkbox :value="key" v-model="activeKeys"
          :class="[classLegend(key), 'legend']">
          <span class="key">{{ key }}</span>
          <span class="value">{{ latest.value.format() }}</span>
        </Checkbox>
      </li>
    </ul>
    <div class="options-wrap">
      <Dropdown ref="durations" name="duration" @change="onDurationChange">
        <DropdownLabel>Duration</DropdownLabel>
        <DropdownItem v-for="(duration, i) in $root.options.durations"
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
      $.$d3.csv(TELESCRIBE_HOST + $.body.csvBox.boundaries)
        .row(function(r) {
          boundaries.push(+r.timestamp);
        })
        .get(undefined, function() {
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
    body: {
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
    fullName() {
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

        var key = newVal[i];
        if($.dataset[key] == undefined) {
          (async function() {
            var p = new Promise(resolve => {
              // Make buf so as not to invoke watchers
              let buf = [];
              $.$d3.csv(TELESCRIBE_HOST + $.body.csvBox.dataMap[key])
              .row(function(r) {
                buf.push({
                  timestamp: +r.timestamp,
                  value: +r.value
                });
              })
              .get(undefined, function() {
                // $set so as to make watchers work
                $.$set($.dataset, key, buf);
                resolve();
              });
            });
            await p;
            activeDataset[key] = $.dataset[key];
            ensure(++i);
          })();
        } else {
          activeDataset[key] = $.dataset[key];
          ensure(++i);
        }
      };

      ensure();
    }
  },
  methods: {
    classLegend(key) {
      var i = this.activeKeys.indexOf(key);
      return i.toSeries();
    },
    status(key) {
      let map = this.body.latestMap;
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
    formatDuration(t) {
      if(t <= 3600) return Math.round(t / 60) + "m";
      else if(t <= 24 * 3600) return Math.round(t / 3600) + "h";
      else return Math.round(t / 86400) + "d";
    },
    onDurationChange(val) {
      this.$refs.graph.duration = val;
    }
  }
}
</script>
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
    <div class="graph-wrap">
      <Graph ref="graph"/>
    </div>
  </article>
</template>

<script>
export default {
  name: "Client",
  async created() {
    await this.update();
  },

  props: {
    body: {
      type: Object
    }
  },
  data() {
    return {
      activeKeys: [],
      dataset: {}
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
    async update() {
      // Clean up first
      this.dataset = {};

      var $ = this;
      // Get boundaries
      var boundaries = [];
      var p = new Promise(resolve => {
        $.$d3.csv(TELESCRIBE_HOST + this.body.csvBox.boundaries)
          .row(function(r) {
            boundaries.push(+r.timestamp);
          })
          .get(undefined, function() {
            resolve();
          });
      });
      //
      await p;
      $.$refs.graph.duration = 6 * 3600;
      $.$refs.graph.boundaries = boundaries;
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
    }
  }
}
</script>
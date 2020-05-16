<template>
  <article class="client">

    <div class="client-header">
      <div class="name-flex">
        <h2 class="name">{{ info.alias }}</h2>
        <div class="menu-wrap">
          <Button class="menu-button" @click="$refs.menu.toggle($event)">
            <font-awesome icon="caret-down"/>
          </Button>
          <Menu ref="menu">
            <MenuItem @click="$refs.modal.open = true">Constants</MenuItem>
          </Menu>
        </div>
      </div>
      <div class="summary-flex">
        <div class="status">
          <Icon :type="statusIconOf(statusMap)" />
        </div>
        <div class="info">{{ id }} &middot; {{ info.host }}</div>
        <Badge v-for="tag in tags" :key="tag">{{ tag }}</Badge>
      </div>
      <Rule type="hr" variant="dark"/>
    </div>

    <Modal ref="modal">
      <h3>Constants</h3>
      <table>
        <thead>
          <tr>
            <th>Item</th><th>Value</th>
          </tr>
        </thead>
        <tbody v-show="configMapReady">
          <tr v-for="(itemStatus, monitorKey) in constantItemStatusMap"
            :key="monitorKey">
            <td>{{ monitorKey }}</td>
            <td>{{ formatMonitorValue(monitorKey, itemStatus.value) }}</td>
          </tr>
        </tbody>
      </table>
    </Modal>

    <Cover v-show="!graphReady">
      No Available Data
    </Cover>
    <div v-show="graphReady" class="client-graph">
        
      <div class="graph-options">
        <div class="option option--keys">
          <Select v-model="activeKeys" multiple>
            <SelectItem v-for="(status, mKey) in statusMap"
              :key="mKey" :value="mKey">
              <Icon :type="statusIconOf(status.status, true)"/> {{ mKey }}
            </SelectItem>
          </Select>
        </div>
        <div class="option option--duration">
          <ButtonGroup>
            <Button v-for="d in $root.webConfig.durations"
              :variant="d === duration ? 'accent' : ''"
              :key="d" @click="duration = d">{{ formatDuration(d) }}</Button>
          </ButtonGroup>
        </div>
      </div>

      <div class="graph-focus-info">
        <div class="focused-time">{{ focusedTime }}</div>
        <div class="focused-values">
          <div class="value"
            v-for="mKey in activeKeys" 
            :key="mKey"
            :style="{color: colorify(mKey)}">
            {{ mKey }}: {{ focusedValue(mKey) }}
          </div>
        </div>
      </div>

      <div class="graph-wrap">
        <Graph ref="graph" :duration="duration" :boundaries="boundaries"/>
      </div>

    </div>

  </article>
</template>

<script>

import {csvParse} from 'd3-dsv';
const d3 = {csvParse};
import {NumberFormatter, statusIconOf, splitWhitespace} from '@/lib/util/web.js';
import {colorify} from '@/lib/ui/util/util.js';
import Queue from '@/lib/util/queue.js';
import {library} from '@fortawesome/fontawesome-svg-core';
import {
  faCaretDown
} from '@fortawesome/free-solid-svg-icons';
library.add(faCaretDown);

export default {
  name: "Client",
  created() {
    let $ = this;

    // Clean up first
    $.dataset = {};
    let boundaries = [];

    let i = 0;

    // Populate Config Map
    for(let monitorKey in this.statusMap) {
      this.queue.queue(async() => {
        let j = i++;
        console.log("start", j);
        let rsp = await $.$api.v1.getMonitorConfig($.id, monitorKey);
        let cfg = rsp.monitorConfig;
        $.configMap[monitorKey] = cfg;
        console.log("end", j);
      });
    }
    this.queue.queue(async() => {
      let j = i++;
      console.log("start", j);
      $.configMapReady = true;
      console.log("end", j);
    });

    // Get Monitor Data Boundaries
    this.queue.queue(async() => {
      let csv = await $.$api.v1.getMonitorDataBoundaries($.id);
      d3.csvParse(csv, row => {
        boundaries.push(+row.timestamp);
      });
      $.boundaries = boundaries;
    });

  },
  mounted() {
    this.mounted = true;
  },

  props: {
    info: Object,
    statusMap: Object
  },
  data() {
    return {
      activeKeys: [],
      boundaries: [],
      duration:   this.$root.webConfig.durations[0],
      dataset:    {},
      configMap:  {},
      configMapReady: false, // TODO: this is temp var
      queue:      new Queue(),
      mounted:    false
    };
  },
  computed: {
    id() {return this.$vnode.key;},
    graphReady() {return this.boundaries.length > 0 && this.mounted;},
    tags() {return splitWhitespace(this.info.tags);},
    
    focusedTime() {
      let fmt = this.$root.webConfig["format.date.long"];
      if(this.graphReady) {
        let graph = this.$refs.graph;
        let arr;
        if(graph.focusedTimestamps.length > 0) arr = graph.focusedTimestamps;
        else if(graph.visibleBoundary)         arr = graph.visibleBoundary;

        if(arr) return arr.map(d => d.date(fmt)).join(" – ");
      }
      return "-";
    },

    constantItemStatusMap() {
      let ret = {};
      for(let monitorKey in this.statusMap) {
        let itemStatus = this.statusMap[monitorKey];
        if(itemStatus.constant)
          ret[monitorKey] = itemStatus;
      }
      return ret;
    }

  },
  watch: {
    graphReady(newVal) {
      if(newVal === true) {
        this.$refs.graph.plot({});
      }
    },
    activeKeys(newVal, oldVal) {
      var $ = this;
      var activeDataset = {};
      var ensure = (i = 0) => {
        if(i >= newVal.length) {
          $.$refs.graph.plot(activeDataset);
          return;
        }

        var mKey = newVal[i];
        (async function() {
          if($.dataset[mKey] == undefined) {
            var p = new Promise(resolve => {
              // Make buf so as not to invoke watchers
              let buf = [];

              $.$api.v1.getMonitorDataTable($.id, mKey)
                .then(function(csv) {
                  d3.csvParse(csv, row => {
                    buf.push({
                      timestamp: +row.timestamp,
                      value: +row.value
                    });
                  });

                  let cfg = $.configMap[mKey];
                  $.$set($.dataset, mKey, {
                    data: buf,
                    color: colorify(mKey),
                    formatter: new NumberFormatter(cfg.format)
                  });

                  resolve();
                });
            });
            await p;
          }
          activeDataset[mKey] = $.dataset[mKey];
          ensure(++i);
        })();
      };

      ensure();
    }
  },
  methods: {
    colorify,
    statusIconOf,
    formatDuration(t) {
      if(t <= 60) return `${t}m`;
      else if(t <= 24 * 60) return Math.round(t / 60) + "h";
      else return Math.round(t / 1440) + "d";
    },
    focusedValue(mKey) {
      if(this.graphReady) {
        let graph = this.$refs.graph;
        let value = graph.focusedValues[mKey];
        if(!(value === undefined)) {
          return Number(value).format(this.configMap[mKey].format);
        }
      }
      return "-";
    },
    formatMonitorValue(monitorKey, value) {
      console.log(this.configMapReady);
      let format = this.configMap[monitorKey].format;
      return Number(value).format(format);
    }
  }
}
</script>
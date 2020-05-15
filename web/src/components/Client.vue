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
        <tbody>
          <tr>
            <td>cpu-count</td><td>4 CPUs</td>
          </tr>
          <tr>
            <td>memory-size-gb</td><td>7.83 GB</td>
          </tr>
          <tr>
            <td>swap-size-gb</td><td>3.92 GB</td>
          </tr>
          <tr>
            <td>disk-size-gb[xvda1]</td><td>49.71 GB</td>
          </tr>
          <tr>
            <td>disk-size-gb[sda1]</td><td>9.66 GB</td>
          </tr>
          <tr>
            <td>command(minecraft-users)</td><td>8 Online</td>
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
            <Button v-for="d in $root.webCfg.durations"
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

    // Get ClientRole
    this.queue.queue(new Promise(resolve => {
      $.$api.v1.getClientRole($.id)
        .then(function(rsp) {
          $.role = rsp.clientRole;
          resolve();
        });
    }));

    // Get Monitor Data Boundaries
    this.queue.queue(new Promise(resolve => {
      $.$api.v1.getMonitorDataBoundaries($.id)
        .then(function(csv) {
          d3.csvParse(csv, row => {
            boundaries.push(+row.timestamp);
          });
          $.boundaries = boundaries;
          resolve();
        });
    }));

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
      duration:   this.$root.webCfg.durations[0],
      dataset:    {},
      configMap:  {},
      queue:      new Queue(),
      mounted:    false,

      temp1: false
    };
  },
  computed: {
    id() {return this.$vnode.key;},
    graphReady() {return this.boundaries.length > 0 && this.mounted;},
    tags() {return splitWhitespace(this.info.role);},
    
    focusedTime() {
      let fmt = this.$root.webCfg["format.date.long"];
      if(this.graphReady) {
        let graph = this.$refs.graph;
        let arr;
        if(graph.focusedTimestamps.length > 0) arr = graph.focusedTimestamps;
        else if(graph.visibleBoundary)         arr = graph.visibleBoundary;

        if(arr) return arr.map(d => d.date(fmt)).join(" – ");
      }
      return "-";
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

                  $.$api.v1.getMonitorConfig($.id, mKey)
                    .then(function(rsp) {
                      let cfg = rsp.monitorConfig;
                      $.configMap[mKey] = cfg;
                      $.$set($.dataset, mKey, {
                        data: buf,
                        color: colorify(mKey),
                        formatter: new NumberFormatter(cfg.format)
                      });

                      resolve();
                    });
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
    colorify(str) {
      return colorify(str);
    }
  }
}
</script>
<template>
  <article class="client">

    <div class="client-header">
      <div class="name-flex">
        <h2 class="name">{{ info.alias }}</h2>
        <div class="menu-wrap">
          <Button class="menu-button" @click="$refs.menu.toggle($event)">
            <font-awesome icon="ellipsis-h"/>
          </Button>
          <Menu ref="menu">
            <MenuItem>Add to Favorites</MenuItem><!-- TODO -->
            <MenuItem @click="$refs.modal.open = true">Constants</MenuItem>
          </Menu>
        </div>
      </div>
      <div class="summary-flex">
        <div class="status">
          <Icon :type="statusIconOf(itemStatusMap)" />
        </div>
        <div class="info">{{ id }} &middot; {{ info.host }}</div>
        <Badge v-for="tag in tags" :key="tag">{{ tag }}</Badge>
      </div>
      <Rule type="hr" variant="dark"/>
    </div>

    <Modal ref="modal">
      <h3>Constants</h3>
      <table><!-- TODO -->
        <tbody>
          <tr v-for="(itemStatus, monitorKey) in itemStatusMap"
            :key="monitorKey" v-if="isConstant(monitorKey)">
            <td>{{ monitorKey }}</td>
            <td>{{ formatNumber(monitorKey, itemStatus.value) }}</td>
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
            <SelectItem v-for="(itemStatus, monitorKey) in itemStatusMap"
              :key="monitorKey" :value="monitorKey"
              v-if="isConstant(monitorKey) === false">
              <Icon :type="statusIconOf(itemStatus.status, true)"/> {{ monitorKey }}
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
            v-for="monitorKey in activeKeys" 
            :key="monitorKey"
            :style="{color: colorify(monitorKey)}">
            {{ monitorKey }}: {{ focusedValue(monitorKey) }}
          </div>
        </div>
      </div>

      <div class="graph-wrap">
        <Graph ref="graph"
          :dataset="activeDataset"
          :duration="duration"
          :boundaries="boundaries"
          :options="graphOptions"
          v-model="focusedDatumMap"/>
      </div>

    </div>

  </article>
</template>

<script>

import moment from 'moment/src/moment';
import {csvParse} from 'd3-dsv';
import {extent} from "d3-array";
const d3 = {csvParse, extent};
import {NumberFormatter, formatNumber, statusIconOf, splitWhitespace} from '@/lib/util/web.js';
import {colorify} from '@/lib/ui/util/util.js';
import Queue from '@/lib/util/queue.js';
import Graph from '@/lib/graph';
import {library} from '@fortawesome/fontawesome-svg-core';
import {
  faEllipsisH
} from '@fortawesome/free-solid-svg-icons';
library.add(faEllipsisH);

export default {
  name: "Client",
  components: {Graph},
  created() {
    let $ = this;

    // Clean up first
    $.dataset = {};
    let boundaries = [];

    // Populate Config Map
    for(let monitorKey in this.itemStatusMap) {
      this.queue.queue(async() => {
        let rsp = await $.$api.v1.getMonitorConfig($.id, monitorKey);
        let cfg = rsp.monitorConfig;
        $.monitorConfigMap[monitorKey] = cfg;
      });
    }

    // Get Monitor Data Boundaries
    this.queue.queue(async() => {
      let csv = await $.$api.v1.getMonitorDataBoundaries($.id);
      d3.csvParse(csv, row => boundaries.push(+row.timestamp));
      $.boundaries = boundaries;
    });

  },

  props: {
    info: Object,
    itemStatusMap: Object
  },
  data() {
    var $ = this;
    return {
      activeKeys: [],
      activeDataset: {},
      boundaries: [],
      duration:   this.$root.webConfig.durations[0],
      focusedDatumMap: {},
      dataset:    {},
      graphOptions: {
        accessors: {
          x: d => d.timestamp,
          y: d => d.value
        },
        formatters: {
          xAxis: x => moment.unix(x).format($.$root.webConfig["format.date.short"]),
          yAxis: y => formatNumber($.$root.webConfig["format.yAxis"], y),
          x: d => moment.unix(d.timestamp).format($.$root.webConfig["format.date.long"]),
          y: d => formatNumber($.$root.webConfig["format.value"], d.value),
        }
      },
      monitorConfigMap:  {},
      queue:      new Queue()
    };
  },
  computed: {
    id() {return this.$vnode.key;},
    tags() {return splitWhitespace(this.info.tags);},
    graphReady() {return this.boundaries.length > 0},
    
    focusedTime() {
      let fmt = this.$root.webConfig["format.date.long"];
      let fdm = this.focusedDatumMap;
      if(this.graphReady && Object.keys(fdm).length > 0) {
        let arr = [];
        for(let monitorKey in fdm) {
          arr.push(fdm[monitorKey].timestamp);
        }
        return d3.extent(arr).map(
          d => moment.unix(d).format(fmt)
        ).join(" – ");
      }
      return "-";
    }

  },
  watch: {

    activeKeys(newVal, oldVal) {

      let $ = this;
      let activeDataset = {};

      this.queue.queue(async() => {

        for(let i = 0; i < newVal.length; i++) {

          let monitorKey = newVal[i];

          if($.dataset[monitorKey] === undefined) {

            let format = $.monitorConfigMap[monitorKey].format;
            let buf = [];
            let csv = await $.$api.v1.getMonitorDataTable($.id, monitorKey);

            // Make this part of documentation
            d3.csvParse(csv, row => buf.push({
              timestamp: +row.timestamp,
              value: +row.value,
              per: +row.per
            }));

            $.$set($.dataset, monitorKey, {
              data: buf,
              color: colorify(monitorKey),
              formatters: {
                y: d => `${formatNumber(format, d.value)} / ${d.per}s`
              }
            });

          }

          activeDataset[monitorKey] = $.dataset[monitorKey];

        }
        
        $.activeDataset = activeDataset;

      });

    }

  },
  methods: {
    colorify,
    statusIconOf,
    isConstant(monitorKey) {
      let monitorConfig = this.monitorConfigMap[monitorKey];
      return monitorConfig ? monitorConfig.constant : undefined;
    },
    formatDuration(t) {
      if(t <= 60) return `${t}m`;
      else if(t <= 24 * 60) return Math.round(t / 60) + "h";
      else return Math.round(t / 1440) + "d";
    },
    formatNumber(monitorKey, value) {
      let monitorConfig = this.monitorConfigMap[monitorKey];
      let format = monitorConfig ? monitorConfig.format : "";
      return formatNumber(format, value);
    },
    focusedValue(monitorKey) {
      if(this.graphReady) {
        let datum = this.focusedDatumMap[monitorKey];
        if(!(datum === undefined)) {
          return this.formatNumber(monitorKey, datum.value);
        }
      }
      return "-";
    }
  }
}
</script>
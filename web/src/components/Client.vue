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
            <MenuItem v-if="!isFavorite" @click="toggleFavorite">Add to Favorites</MenuItem>
            <MenuItem v-else @click="toggleFavorite">Cancel Favorite</MenuItem>
            <MenuItem @click="$refs.constantsModal.open = true">Constants</MenuItem>
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

    <Modal class="constants-modal" ref="constantsModal">
      <h4>Constants</h4>
      <table class="constants-table">
        <tbody>
          <tr v-for="(itemStatus, monitorKey) in itemStatusMap"
            :key="monitorKey" v-if="isConstant(monitorKey)">
            <td>{{ monitorKeyAlias(monitorKey) }}</td>
            <td>{{ formatDatum(monitorKey, itemStatus) }}</td>
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
            <Button v-for="d in webConfig.durations"
              :variant="d === duration ? 'success' : ''"
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
            {{ monitorKeyAlias(monitorKey) }}: {{ focusedValue(monitorKey) }}
          </div>
        </div>
      </div>

      <div class="graph-wrap">
        <Graph ref="graph"
          :dataset="activeDataset"
          :duration="duration * 60"
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

  props: {
    info: Object,
    itemStatusMap: Object
  },

  created() {

    let $ = this;
    let boundaries = [];
    
    // Vars
    this.duration = this.webConfig.durations[0];

    // Populate Config Map
    for(let monitorKey in this.itemStatusMap) {
      this.queue.queue(async() => {
        let {monitorConfig} = await $.$api.v1.getMonitorConfig($.id, monitorKey);
        $.monitorConfigMap[monitorKey] = monitorConfig;
      });
    }

    // Get Monitor Data Boundaries
    this.queue.queue(async() => {
      let csv = await $.$api.v1.getMonitorDataBoundaries($.id);
      d3.csvParse(csv, row => boundaries.push(+row.timestamp));
      $.boundaries = boundaries;
    });

  },

  data() {
    return {
      activeKeys:       [],
      activeDataset:    {},
      boundaries:       [],
      duration:         undefined,
      focusedDatumMap:  {},
      dataset:          {},
      monitorConfigMap: {},
      queue: new Queue()
    };
  },

  computed: {

    isFavorite() {return this.$parent.isFavorite(this.id)},

    id()         {return this.$vnode.key},
    tags()       {return splitWhitespace(this.info.tags)},
    graphReady() {return this.boundaries.length > 0},
    graphOptions() {
      var $ = this;
      return {
        accessors: {
          x: d => d.timestamp,
          y: d => d.value
        },
        formatters: {
          xAxis: x => moment.unix(x).format($.webConfig["format.date.short"]),
          yAxis: y => formatNumber($.webConfig["format.yAxis"], y),
          x:     d => moment.unix(d.timestamp).format($.webConfig["format.date.long"]),
          y:     d => formatNumber($.webConfig["format.value"], d.value)
        }
      };
    },
    
    focusedTime() {
      let fmt = this.$root.webConfig["format.date.long"];
      let fdm = this.focusedDatumMap;
      if(this.graphReady && Object.keys(fdm).length > 0) {
        let arr = [];
        for(let monitorKey in fdm) {
          arr.push(fdm[monitorKey].timestamp);
        }
        arr = d3.extent(arr);
        arr = arr[0] === arr[1] ? [arr[0]] : arr;
        
        return arr.map(
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
            let filter = {
              from: 0, to: 2100000000, per: 1
            };
            let csv = await $.$api.v1.getMonitorDataCsv($.id, monitorKey, filter);

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
                y: d => `${$.formatDatum(monitorKey, d)}`
              }
            });

          }

          activeDataset[monitorKey] = $.dataset[monitorKey];

        }
        
        $.activeDataset = activeDataset;
        console.log(activeDataset);

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
    monitorKeyAlias(monitorKey) {
      let {alias} = this.monitorConfigMap[monitorKey];
      if(alias && alias !== "") return alias;
      return monitorKey;
    },
    formatDatum(monitorKey, datum) {
      let monitorConfig = this.monitorConfigMap[monitorKey];
      let format = monitorConfig ? monitorConfig.format : "";
      let ret = formatNumber(format, datum.value);
      if(monitorConfig.absolute === false)
        ret += `∕${datum.per}s`;
      return ret;
    },
    focusedValue(monitorKey) {
      if(this.graphReady) {
        let datum = this.focusedDatumMap[monitorKey];
        if(!(datum === undefined)) {
          return this.formatDatum(monitorKey, datum);
        }
      }
      return "-";
    },
    toggleFavorite() {
      this.$parent.toggleFavorite(this.id);
    }
  }
}
</script>
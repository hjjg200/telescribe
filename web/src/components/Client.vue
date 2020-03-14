<template>
  <article class="client">
    <div class="temp1">
      <h2 class="name">{{ info.alias }}</h2>
      <div class="status-flex">
        <div class="light"></div>
        <div class="info">{{ id }} &middot; {{ info.host }}</div>
        <div class="tags">
          <span class="role-tag">foo</span>
          <span class="role-tag">bar</span>
        </div>
      </div>
      <hr class="dark"/>
    </div>

    <div class="temp2">
      <div class="left">
        <h3>System</h3>
      </div>

      <div class="right">
        <table>
          <tbody>
            <tr>
              <td><font-awesome icon="microchip"/></td>
              <td>CPU</td>
              <td>2</td>
            </tr>
            <tr>
              <td><font-awesome icon="memory"/></td>
              <td>Swap</td>
              <td>3 GB</td>
            </tr>
            <tr>
              <td><font-awesome icon="server"/></td>
              <td>OS</td>
              <td>Ubuntu 18.04</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    
    <div class="card">
      <div class="card__section card__ui-test">
        <div class="frame">
          <Checkbox value="a" v-model="fruit"><Icon type="error"/> Apple</Checkbox>
          <Checkbox value="b" v-model="fruit"><Icon type="warning"/> Banana</Checkbox>
          <Checkbox value="c" v-model="fruit"><Icon type="green-light"/> Coconut</Checkbox>
        </div>
        <div class="frame">
          <Button>Button 1</Button>
          <Button type="accent">Button 2</Button>
        </div>
        <div class="frame" style="line-height: 1.5;">
          <Select v-model="fruit2" multiple>
            <SelectItem value="a">Apple</SelectItem>
            <SelectItem value="b">Banana</SelectItem>
            <SelectItem value="c">Coconut</SelectItem>
          </Select>
        </div>
        <div class="frame">
          <Dropdown v-model="fruit2">
            <DropdownItem>Apple</DropdownItem>
            <DropdownItem>Banana</DropdownItem>
            <DropdownItem>Coconut</DropdownItem>
          </Dropdown>
        </div>
      </div>
    </div>

    <div class="graph-options">
      <div class="option option--keys">
        <Select v-model="activeKeys" multiple>
          <SelectItem v-for="(status, mKey) in statusMap"
            :key="mKey" :value="mKey">
            <Icon :type="iconTypeOf(status.status)"/> {{ mKey }}
          </SelectItem>
        </Select>
      </div>
      <div class="option option--duration">
        <ButtonGroup>
          <Button v-for="d in $root.webCfg.durations"
            :type="d === duration ? 'accent' : ''"
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



  </article>
</template>

<script>

import {csvParse} from 'd3-dsv';
const d3 = {csvParse};
import {NumberFormatter} from '@/lib/util/web.js';
import {colorify} from '@/lib/ui/util/util.js';
import Queue from '@/lib/util/queue.js';
import {library} from '@fortawesome/fontawesome-svg-core';
import {
  faMicrochip, faHdd, faServer, faMemory
} from '@fortawesome/free-solid-svg-icons';
library.add(faMicrochip, faHdd, faServer, faMemory);

export default {
  name: "Client",
  created() {
    let $ = this;

    // Clean up first
    $.dataset = {};
    let boundaries = [];

    this.queue.queue(new Promise(resolve => {
      $.$api.v1.getClientStatus($.id)
        .then(function(rsp) {
          $.statusMap = rsp.clientStatus;
          resolve();
        });
    }));
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
    info: {type: Object}
  },
  data() {
    return {
      activeKeys: [],
      boundaries: [],
      duration:   this.$root.webCfg.durations[0],
      dataset:    {},
      queue:      new Queue(),
      statusMap:  {},
      mounted:    false,

      fruit: [],
      fruit2: [],
      fruit3: [],
      temp1: null
    };
  },
  computed: {
    id() {return this.$vnode.key;},
    graphReady() {return this.boundaries.length > 0 && this.mounted;},
    focusedTime() {
      let fmt = this.$root.webCfg["format.date.long"];
      if(this.graphReady) {
        let graph = this.$refs.graph;
        if(graph.focusedTime)     return graph.focusedTime.date(fmt);
        if(graph.visibleBoundary) return graph.visibleBoundary.map(d => d.date(fmt)).join(" – ");
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
                  $.$set($.dataset, mKey, {
                    data: buf,
                    color: colorify(mKey),
                    formatter: new NumberFormatter("{.2f}")
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
    iconTypeOf(status) {
      if(status === 8) return 'warning';
      else if(status === 16) return 'error';
    },
    formatDuration(t) {
      if(t <= 60) return `${t}m`;
      else if(t <= 24 * 60) return Math.round(t / 60) + "h";
      else return Math.round(t / 1440) + "d";
    },
    focusedValue(mKey) {
      if(this.graphReady) {
        let graph = this.$refs.graph;
        let value = graph.focusedValues[mKey];
        if(value) return value;
      }
      return "-";
    },
    colorify(str) {
      return colorify(str);
    }
  }
}
</script>
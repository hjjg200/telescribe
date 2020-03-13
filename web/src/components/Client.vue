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
          <Checkbox value="a" v-model="fruit"><Icon type="error"/>Apple</Checkbox>
          <Checkbox value="b" v-model="fruit"><Icon type="warning"/>Banana</Checkbox>
        </div>
        <div class="frame">
          <Button>Button 1</Button>
          <Button class="ui-button--accent">Button 2</Button>
        </div>
        <div class="frame">
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


    <div class="card client-header">
      <div class="card__section left">
        <div class="left__top">
          <h2>{{ info.alias }}</h2>
          <p>{{ id }} &middot; {{ info.host }}</p>
        </div>
        <div class="left__bottom">
          <ul>
            <li>Role: foo bar</li>
            <li>Last Access: Feb 29</li>
          </ul>
        </div>
      </div>
      <div class="card__section right">
        <h4>System</h4>
        <ul class="system">
          <li>
            <span class="type"><font-awesome icon="microchip"/> CPU</span>
            <span class="value">1</span>
          </li>
          <li>
            <span class="type"><font-awesome icon="memory"/> Swap</span>
            <span class="value">2 GB</span>
          </li>
          <li>
            <span class="type"><font-awesome icon="server"/> OS</span>
            <span class="value">Ubuntu 18.04</span>
          </li>
        </ul>
      </div>
    </div>

    <div class="card client-graph">

      <div class="card__section card__legend">
        <ul class="checkboxes">
          <li v-for="(status, mKey) in statusMap"
            :key="mKey" :data-status="status.status">
            <Checkbox :value="mKey" v-model="activeKeys"
              :markClass="[classLegend(mKey), 'legend']">
              <span class="key">{{ mKey }}</span>
              <span class="value">{{ status.value.format("{.2f}") }}</span>
            </Checkbox>
          </li>
        </ul>
      </div>

      <div class="card__section card__graph-options">
        <div class="options-wrap">
          <div class="option">
            <label>Duration</label>
            <Select ref="durations" name="duration" v-model="duration">
              <SelectItem v-for="(duration, i) in $root.webCfg.durations"
                :key="i"
                :value="duration">{{ formatDuration(duration) }}</SelectItem>
            </Select>
          </div>
        </div>
      </div>

      <div class="card__section card__graph">
        <div class="graph-wrap">
          <Graph ref="graph" :duration="duration" :boundaries="boundaries"/>
        </div>
      </div>

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
    let $ = this;
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
    graphReady() {return this.boundaries.length > 0 && this.mounted;}
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
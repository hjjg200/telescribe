<template>
  <article class="client">
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
            <Dropdown ref="durations" name="duration" @change="onDurationChange">
              <DropdownItem v-for="(duration, i) in $root.webCfg.durations"
                :key="i" :value="duration"
                :selected="i == 0">{{ formatDuration(duration) }}</DropdownItem>
            </Dropdown>
          </div>
        </div>
      </div>

      <div class="card__section card__graph">
        <div class="graph-wrap">
          <Graph ref="graph"/>
        </div>
      </div>

    </div>

    <div class="card">
      <div class="card__section color-test">
        <p class="secondary">Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean auctor velit diam, eu tempor est commodo sit amet. Vestibulum porttitor volutpat ullamcorper. Etiam euismod ultrices.</p>
        <Button class="secondary">Read More</Button>
        <nav class="secondary">
          <hr class="secondary">
        </nav>
        <hr class="secondary"/>

        <p class="accent">Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean auctor velit diam, eu tempor est commodo sit amet. Vestibulum porttitor volutpat ullamcorper. Etiam euismod ultrices.</p>
        <Button class="accent">Read More</Button>
        <nav class="accent">
          <hr class="accent">
        </nav>
        <hr class="accent"/>

        <p class="secondary-dark">Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean auctor velit diam, eu tempor est commodo sit amet. Vestibulum porttitor volutpat ullamcorper. Etiam euismod ultrices.</p>
        <Button class="secondary-dark">Read More</Button>
        <nav class="secondary-dark">
          <hr class="secondary-dark">
        </nav>
        <hr class="secondary-dark"/>

        <p class="accent-bland">Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean auctor velit diam, eu tempor est commodo sit amet. Vestibulum porttitor volutpat ullamcorper. Etiam euismod ultrices.</p>
        <Button class="accent-bland">Read More</Button>
        <nav class="accent-bland">
          <hr class="accent-bland">
        </nav>
        <hr class="accent-bland"/>
      </div>
    </div>


  </article>
</template>

<script>

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
          $.$d3.csvParse(csv, row => {
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
      dataset:    {},
      queue:      new Queue(),
      statusMap:  {},
      mounted:    false
    };
  },
  computed: {
    id() {return this.$vnode.key;},
    graphReady() {return this.boundaries.length > 0 && this.mounted;}
  },
  watch: {
    graphReady(newVal) {
      if(newVal === true) {
        this.$refs.graph.boundaries = this.boundaries;
        this.$refs.durations.selectIndex(0);
      }
    },
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
                  });
                  $.$set($.dataset, mKey, buf);
                  resolve();
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
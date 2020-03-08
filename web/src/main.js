
import * as d3 from '@/lib/d3.v4.js';
import * as moment from '@/lib/moment.js';

import Vue from 'vue';
import App from './App.vue';

// Personal Lib
import UI from '@/lib/ui';
Vue.use(UI);

// API
import * as api from '@/lib/api';

// Prototype
Vue.prototype.$api = api;
Vue.prototype.$d3 = d3;
Vue.prototype.$moment = moment;

// Utils
import {formatNumber} from '@/lib/util/web.js';

Number.prototype.date = function(str) {
  if(str === undefined) str = "MMM DD HH:mm";
  return moment.unix(this).format(str);
};

Number.prototype.format = function(fmt) {
  return formatNumber(this, fmt);
}
let n = 1.503;
console.log(n.format("{}"));
console.log(n.format("{.4f}"));
console.log(n.format("{.f}"));
console.log(n.format("{f}%"));
console.log(n.format("\\{{f}\\}"));
console.log(n.format("{2.0x}"));
console.log(n.format("{3x.f}"));


function _temp1(str) {
  if(str === "" || str === undefined) str = "{}";
  var fmt = parseNumberFormat(str);
  var num = this;
  num = num * Math.pow(10, fmt.exp);
  // Precision
  if(!isNaN(fmt.precision)) {
    var precision = Math.pow(10, fmt.precision);
    num = Math.round(num * precision) / precision;
  }
  return `${fmt.prefix}${formatComma(num)}${fmt.suffix}`;
};

Number.prototype.toSeries = function() {
  return "series-" + "abcdefghijklmnopqrstuvwxyz".charAt(this);
};

String.prototype.escapeQuote = function() {
  return this.replace(/"/g, '\\\"');
};

Element.prototype.hasClass = function(className) {
  return (' ' + this.getAttribute('class') + ' ').indexOf(' ' + className + ' ') > -1;
}

function getSeriesIdx(i) {
  return "abcdefghijklmnopqrstuvwxyz".charAt(i - 1);
}

function parseNumberFormat(str) {
  var rgx = /(.+)?(\{(e([+-]?\d+))?(\.(\d+))?f?\})(.+)?/;
  var m = str.match(rgx);
  var [prefix = "", exp = 0, precision = 2, suffix = ""] = [m[1], m[4], m[6], m[7]];
  return {
    prefix, exp: parseInt(exp), precision: parseInt(precision), suffix
  };
}

function formatComma(x) {
  var parts = x.toString().split(".");
  parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return parts.join(".");
}

// MAIN
(async function() {
  let clMap   = (await api.v1.getClientMap()).clientMap;
  let webCfg  = (await api.v1.getWebConfig()).webConfig;
  let version = (await api.v1.getVersion()).version;

  console.log(webCfg);

  new Vue({
    data: {clMap, webCfg, version},
    created() {
      document.title = "Telescribe";
    },
    render: h => h(App)
  }).$mount("#app");
})();

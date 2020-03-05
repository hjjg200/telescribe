
import * as d3 from '@/lib/d3.v4.js';
import * as moment from '@/lib/moment.js';

import Vue from 'vue';
import App from './App.vue';

// Personal Lib
import UI from '@/lib/ui';

Vue.use(UI);

// API
import * as api from '@/lib/api';

(async function() {
  try {
    console.log(await api.v1.getVersion());
    console.log(await api.v1.getClientMap());
  } catch(apiErr) {
    console.error(apiErr.toString());
  }
})();

// Prototype
Vue.prototype.$api = api;
Vue.prototype.$d3 = d3;
Vue.prototype.$moment = moment;

// Global

// + Host for telescribe resources
// + empty host is the same host as web
// + this variable is for development purpose to separate npm server and telescribe server
window.TELESCRIBE_HOST = process.env.VUE_APP_TELESCRIBE_HOST;

//

Number.prototype.date = function(str) {
  if(str === undefined) str = "DD HH:mm";
  return moment.unix(this).format(str);
};

Number.prototype.format = function(str) {
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

async function fetchJson(url) {
  var response = await fetch(url, {
    method: "GET"
  });
  console.log(response.status);
  return await response.json();
}

async function fetchText(url) {
  var response = await fetch(url, {
    method: "GET"
  });
  return await response.text();
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
  var abstract = await fetchJson(TELESCRIBE_HOST + "/abstract.json");
  var options = await fetchJson(TELESCRIBE_HOST + "/options.json");
  var version = await fetchText(TELESCRIBE_HOST + "/version");

  new Vue({
    data: {
      abstract: abstract,
      options: options,
      version: version
    },
    created: function() {
      document.title = "Telescribe";
    },
    render: h => h(App)
  }).$mount("#app");
})();

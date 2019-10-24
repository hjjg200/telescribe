
import * as d3 from '@/lib/d3.v4.js';
import * as moment from '@/lib/moment.js';

import Vue from 'vue';
import App from './App.vue';

Vue.prototype.$d3 = d3;
Vue.prototype.$moment = moment;

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
  return "series-" + "abcdefghijklmno".charAt(this);
};

String.prototype.escapeQuote = function() {
  return this.replace(/"/g, '\\\"');
};

Element.prototype.hasClass = function(className) {
  return (' ' + this.getAttribute('class') + ' ').indexOf(' ' + className + ' ') > -1;
}

async function fetchJson(url) {
  var promise = await fetch(url, {
    method: "GET",
    cache: "no-cache"
  });
  return await promise.json();
}

function getSeriesIdx(i) {
  return "abcdefghijklmno".charAt(i - 1);
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
  var abstract = await fetchJson("/abstract.json");
  var options = await fetchJson("/options.json");

  new Vue({
    data: {
      abstract: abstract,
      options: options
    },
    render: h => h(App)
  }).$mount("#app");
})();

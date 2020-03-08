
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

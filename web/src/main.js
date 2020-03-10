
import moment from 'moment/src/moment';

import Vue from 'vue';
import App from './App.vue';

// Personal Lib
import UI from '@/lib/ui';
Vue.use(UI);

// API
import * as api from '@/lib/api';

// Prototype
Vue.prototype.$api = api;

// Utils
import {NumberFormatter} from '@/lib/util/web.js';

Number.prototype.date = function(str) {
  if(str === undefined) str = "MMM DD HH:mm";
  return moment.unix(this).format(str);
};

Number.prototype.format = function(fmt) {
  return (new NumberFormatter(fmt)).format(this);
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

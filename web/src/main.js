
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

String.prototype.escapeQuote = function() {
  return this.replace(/"/g, '\\\"');
};

Element.prototype.hasClass = function(className) {
  return (' ' + this.getAttribute('class') + ' ').indexOf(' ' + className + ' ') > -1;
}



// MAIN
(async function() {
  let clMap   = (await api.v1.getClientMap()).clientMap;
  let webCfg  = (await api.v1.getWebConfig()).webConfig;
  let version = (await api.v1.getVersion()).version;

  NumberFormatter.defaultFormat(webCfg['format.value']);
  console.log(0.00123.format());
  console.log(1.02.format());
  console.log(123.0.format());
  console.log(1234.0.format());
  console.log(999000.0.format());
  console.log(1.234e+15.format());

  let clStatMap = {};
  for(let id in clMap) {
    try {
      clStatMap[id] = (await api.v1.getClientStatus(id)).clientStatus;
    } catch(ex) {
      continue;
    }
  }

  new Vue({
    data: {clMap, clStatMap, webCfg, version},
    created() {
      document.title = "Telescribe";
    },
    render: h => h(App)
  }).$mount("#app");
})();

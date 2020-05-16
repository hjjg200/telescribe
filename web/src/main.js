
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


// MAIN
(async function() {
  let infoMap   = (await api.v1.getClientInfoMap()).clientInfoMap;
  let webConfig = (await api.v1.getWebConfig()).webConfig;
  let version   = (await api.v1.getVersion()).version;

  // Set the default format
  NumberFormatter.defaultFormat(webConfig['format.value']);

  let itemStatusMap = {};
  for(let id in infoMap) {
    try {
      itemStatusMap[id] = (await api.v1.getClientItemStatus(id)).clientItemStatus;
    } catch(ex) {
      continue;
    }
  }

  new Vue({
    data: {
      infoMap, itemStatusMap, webConfig, version
    },
    created() {
      document.title = "Telescribe";
    },
    render: h => h(App)
  }).$mount("#app");
})();

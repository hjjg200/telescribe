
import Vue from 'vue';
import App from './App.vue';

// UI Lib
import UI from '@/lib/ui';
Vue.use(UI);

// API
import * as api from '@/lib/api';
Vue.prototype.$api = api;

// Utils
import {NumberFormatter} from '@/lib/util/web.js';

// MAIN
(async function() {

  let {webConfig} = await api.v1.getWebConfig();
  let {version}   = await api.v1.getVersion();

  // Set the default format
  NumberFormatter.defaultFormat(webConfig['format.value']);

  // Global variables
  Vue.mixin({
    data() {
      return {webConfig, version};
    }
  });

  // Create
  new Vue({
    created() {
      document.title = "Telescribe";
    },
    render: h => h(App)
  }).$mount("#app");

})();

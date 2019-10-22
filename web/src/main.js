import './include/d3.v4.js';
import './include/moment.js';
import './include/helper.js';

import Vue from 'vue';
import App from './App.vue';

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

import {Client} from "./module/client.js";
import {Chart} from "./module/chart.js";

let app;
let promise = preProcess();
let clients = {};

/*

the darn javascript compatibility taught me

- safari ios atm does not support static variables in classes, it gave me unexpected "=" (expecting ( before method)
- while desktop chrome does not require crossorigin in the script tag, safari required it

*/

document.addEventListener("DOMContentLoaded", async function() {

  await promise;

  app = new Vue({
    el: "#app",
    data: {
      abstract: undefined,
      options: undefined,
      clients: clients
    },
    computed: {},
    created: function() {
      this.fetch();
    },
    mounted: function() {
      for(let fullName in this.clients) {
        this.clients[fullName].render();
      }
    },
    methods: {
      fetch: async function() {
        this.abstract = await fetchJson("/abstract.json");
        this.options = await fetchJson("/options.json");
      },
      shortDate: function(t) {
        if(t <= 24 * 3600) return Math.round(t / 3600) + "h";
        else return Math.round(t / 86400) + "d";
      }
    }
  });

});

async function preProcess() {
  var abs = await fetchJson("/abstract.json");
  var opt = await fetchJson("/options.json");

  // Options
  Chart.options(opt);

  // Info
  for(var fullName in abs.clientMap) {
    var client = abs.clientMap[fullName];
    clients[fullName] = new Client(fullName, client);
    console.log(client);
  }

  return;
}
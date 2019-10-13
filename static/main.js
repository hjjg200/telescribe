import {Client} from "./module/client.js";
import {Chart} from "./module/chart.js";

let g_gdc;
let app;
let promise = processGraphDataCompositeV2();
let clients = {};
let rawDataset = {};

/*

the darn javascript compatibility taught me

- safari ios atm does not support static variables in classes, it gave me unexpected "=" (expecting ( before method)
- while desktop chrome does not require crossorigin in the script tag, safari required it

*//*

New graph data composite structure

{
  "clients": {
    "fullName": {
      "monitorData": {
        "key": {
          "format": "..."
          "status": 8
          "csv": "..."
        }
      }
    }
  },
  "options": ...
}

*/

document.addEventListener("DOMContentLoaded", async function() {

  await promise;

  app = new Vue({
    el: "#app",
    data: {
      clients: clients
    },
    computed: {},
    created: function() {},
    mounted: function() {
      for(let fullName in this.clients) {
        this.clients[fullName].render();
      }
    },
    methods: {
      shortDate: function(t) {
        if(t <= 24 * 3600) return Math.round(t / 3600) + "h";
        else return Math.round(t / 86400) + "d";
      }
    }
  });

});

async function processGraphDataCompositeV2() {
  var gdc = await fetchJson("/graphDataCompositeV2.json");

  // Options
  Chart.options(gdc.options);

  // Info
  for(var fullName in gdc.clients) {
    var info = gdc.clients[fullName].monitorData;
    clients[fullName] = new Client(fullName, info);
    // Prepare
    rawDataset[fullName] = {};
    for(var key in info) {
      rawDataset[fullName][key] = [];
    }
  }

  // Dataset
  var p = new Promise(resolve => {
    d3.csv("/monitorData.csv")
    .row(function(r) {
      rawDataset[r.FullName][r.Key].push({
        Timestamp: Number(r.Timestamp),
        Value: Number(r.Value)
      });
    })
    .get(undefined, function() {
      resolve();
    });
  });

  await p;
  // Assign
  for(var fullName in clients) {
    clients[fullName].rawDataset = rawDataset[fullName];
  }
}

async function fetchAndUpdate() {
  
  let gdcResponse = await fetch(
    "graphDataComposite.json", {
    method: "GET",
    cache: "no-cache"
  });

  let cmsResponse = await fetch(
    "clientMonitorStatus.json", {
    method: "GET",
    cache: "no-cache"
  });

  g_gdc = await gdcResponse.json();
  g_cms = await cmsResponse.json();

}
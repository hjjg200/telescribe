import {Chart} from "./chart.js";

export class Client {

  constructor(fullName, info) {
    this.fullName = fullName;
    this.info = info;
    this.keys = Object.keys(info);
    this.activeKeys = [];
  }

  status(key) {
    if(key === undefined) {
      let max = -1;
      for(let key in this.info) {
        let st = this.info[key].status;
        max = st > max ? st : max;
      }
      return max;
    }
    return this.info[key].status;
  }

  render() {
    var dataset = Chart.processDataset(this.rawDataset);
    var ch = new Chart(this.select(".chart"), dataset);
    //
    var $ = this;
    this.chart = ch;
    this.checkboxes = function() {
      var obj = {};
      $.keys.forEach(function(key) {
        obj[key] = $.select(`input[type="checkbox"][value="${key.escapeQuote()}"]`);
      });
      return obj;
    }();
  }

  update() {
    for(let key in this.checkboxes) {
      let box = this.checkboxes[key];
      var i = this.activeKeys.indexOf(box.value);
      if(i === -1) box.className = "";
      else box.classList.add(i.toSeries());
    }
    this.chart.keys(this.activeKeys);
  }

  _li() {
    return document.querySelector(`li[data-fullName="${this.fullName.escapeQuote()}"]`);
  }
  select(q) {
    return this._li().querySelector(q);
  }
  selectAll(q) {
    return this._li().querySelectorAll(q);
  }

}
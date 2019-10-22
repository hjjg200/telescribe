import {Chart} from "./chart.js";

export class Client {

  constructor(fullName, abs) {
    this.fullName = fullName;
    this.abs = abs;
    this.keys = Object.keys(abs.latestMap);
    this.activeKeys = [];
  }

  status(key) {
    let ltMap = this.abs.latestMap;
    if(key === undefined) {
      let max = -1;
      for(let key in ltMap) {
        let st = ltMap[key].status;
        max = st > max ? st : max;
      }
      return max;
    }
    return ltMap[key].status;
  }

  render() {
    var $ = this;
    this.chart = new Chart(this.select(".chart"), this.abs.csvBox);
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
      else box.className = i.toSeries();
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
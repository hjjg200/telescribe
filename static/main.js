
let chartList = {};
let chartLabelsMap = {};
let chartSeriesMap = {};
let chartActiveSeriesMap = {};
let g_gdc;
let g_cms;
let app;

// Put average values instead of null

document.addEventListener("DOMContentLoaded", async function() {

    await fetchAndUpdate();

    app = new Vue({
        el: "#app",
        data: {
            checked: {}
        },
        computed: {
            gdc: function() { return g_gdc },
            cmdMap: function() { return this.gdc.clientMonitorData },
            cmsMap: function() { return g_cms },
            options: function() {
                let gdc = this.gdc;
                let ret = {};
                for(let key in gdc) {
                    if(key.startsWith("options.")) {
                        let split = key.split(".");
                        ret[split[1]] = gdc[key];
                    }
                }
                return ret;
            }
        },
        created: function() {
            for(let fullName in this.cmsMap) {
                this.checked[fullName] = {};
            }
        },
        mounted: function() {
            let options = this.options
            for(let fullName in this.cmdMap) {
                let [labels, seriesMap] = processClientMonitorData(
                    this.cmdMap[fullName], this.cmsMap[fullName], options
                );
                chartLabelsMap[fullName] = labels;
                chartSeriesMap[fullName] = seriesMap;
                chartActiveSeriesMap[fullName] = [];
                this.drawChart(fullName, labels);
            }
        },
        methods: {
            toggleSeries: function(ev, fullName, key, toggle) {
                if(toggle) {
                    chartActiveSeriesMap[fullName].push({
                        key: key,
                        series: chartSeriesMap[fullName][key]
                    });
                    let i = getSeriesIdx(chartActiveSeriesMap[fullName].length);
                } else {
                    let tmp = [];
                    chartActiveSeriesMap[fullName].forEach(function(val) {
                        if(val.key == key) {
                            on = true;
                            return;
                        }
                        tmp.push(val);
                    });
                    chartActiveSeriesMap[fullName] = tmp;
                    ev.target.className = "";
                }
                this.updateChart(fullName);
            },
            updateChart: function(fullName) {
                let series = [];
                let i = 1;
                chartActiveSeriesMap[fullName].forEach(function(val) {
                    let q = formatCheckboxQuery(fullName, val.key);
                    let a = getSeriesIdx(i++);
                    let cb = document.querySelector(q);
                    cb.className = "";
                    cb.classList.add("series-" + a);
                    series.push(val.series);
                });
                chartList[fullName].data.series = series;
                chartList[fullName].update();
                document.querySelectorAll('.ct-point').forEach(function(el) {
                    if(el.getAttribute("ct:meta") == "") el.remove();
                });
            },
            maxStatus: function(cms) {
                let max = -1;
                for(let key in cms) {
                    let st = cms[key].Status;
                    max = st > max ? st : max;
                }
                return max;
            },
            drawChart: function(fullName, labels) {
                let query = formatChartQuery(fullName);
                chartList[fullName] = new Chartist.Line(
                    query, {
                        series: [],
                        labels: labels
                    }, {
                        showPoint: true,
                        fullWidth: true,
                        lineSmooth: false,
                        axisX: {
                            showGrid: false,
                            showLabel: true
                        },
                        axisY: {
                            labelInterpolationFnc: (v, idx) => {
                                if(v >= 1e+9 * 0.8) {
                                    return (Math.round(v / 1e+9 * 10) / 10) + "B";
                                } else if(v >= 1e+6 * 0.8) {
                                    return (Math.round(v / 1e+6 * 10) / 10) + "M";
                                } else if(v >= 1e+3 * 0.8) {
                                    return (Math.round(v / 1e+3 * 10) / 10) + "K";
                                } else {
                                    return v;
                                }
                            }
                        },
                        plugins: [
                            Chartist.plugins.tooltip()
                        ]
                    }
                );
            }
        }
    })

    //
    document.querySelectorAll(".chart").forEach(function(el) {
        el.addEventListener("mouseover", function(ev) {
            if(hasClass(ev.target, "ct-point")) {
                let series = ev.target.parentElement.getAttribute('class').match(/ct-series-\w/)[0];
                let g = document.querySelector("g." + series);
                g.classList.add("active");
            }
        });
        el.addEventListener("mouseout", function(ev) {
            if(hasClass(ev.target, "ct-point")) {
                let series = ev.target.parentElement.getAttribute('class').match(/ct-series-\w/)[0];
                let g = document.querySelector("g." + series);
                g.classList.remove("active");
            }
        });
    });

})


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

function getSeriesIdx(i) {
    return "abcdefghijklmno".charAt(i - 1);
}

function formatChartQuery(fullName) {
    fullName = fullName.replace(/"/g, '\\\"');
    return `#host-list li[data-host="${fullName}"] .chart`;
}

function formatCheckboxQuery(fullName, key) {
    fullName = fullName.replace(/"/g, '\\\"');
    key = key.replace(/"/g, '\\\"');
    return `#host-list li[data-host="${fullName}"] li[data-key="${key}"] input`;
}

function hasClass(element, className) {
    return (' ' + element.getAttribute('class') + ' ').indexOf(' ' + className + ' ') > -1;
}

function processClientMonitorData(cmd, cms, options) {

    let gtht = options.gapThresholdTime * 60;
    let xAxis = [];
    let seriesMap = {};
    { // Always add the current node
        for(let key in cmd) {
            let len = cmd[key].length;
            let st = cms[key];
            let last = cmd[key][len - 1];
            if(len == 0
            || (last != null && last.Timestamp != st.Timestamp)) {
                cmd[key].push({
                    Timestamp: st.Timestamp,
                    Value: st.Value
                });
            }
        }
    }

    let rcMap = {};
    { // Get all timestamps
        let tmp = {};
        for(let key in cmd) {
            let slice = cmd[key];
            rcMap[key] = {};
            for(let idx in slice) {
                let each = slice[idx];
                tmp[each.Timestamp] = null;
                rcMap[key][each.Timestamp] = {
                    value: each.Value,
                    meta: moment.unix(each.Timestamp).format(options.momentJsFormat)
                };
            }
        }
        for(let t in tmp) {
            xAxis.push(t);
        }
        xAxis.sort();
    }

    // Put gaps
    let tmp = [];
    let glen = Math.round(xAxis.length * options.gapPercent / 100) + 1;
    let labelStep = Math.floor(xAxis.length / 5);
    let gapAdded = false;
    let labels = [];
    for(let i = 0; i < xAxis.length; i++) {
        let ts = xAxis[i];
        let nextTs = xAxis[i + 1];
        if(nextTs != null && nextTs - ts > gtht) {// Gap
            gapAdded = true;
            labels.push(getHours(ts));
            tmp.push(ts);
            for(let j = 0; j < glen; j++) {
                labels.push("");
                tmp.push(null);
            }
        } else if(i % labelStep == 0 || gapAdded) {
            gapAdded = false;
            labels.push(getHours(ts));
            tmp.push(ts);
        } else {
            labels.push("");
            tmp.push(ts);
        }
    }
    xAxis = tmp;

    // Series
    for(let key in cmd) {
        let slice = cmd[key];
        let tmp = rcMap[key];
        let series = [];

        for(let i = 0; i < xAxis.length; i++) {

            let aX = xAxis[i];
            let bX = (i < xAxis.length - 1) ? xAxis[i + 1] : null;

            if(aX != null
            && bX != null
            && tmp[aX] != null) {
                
                let aI = tmp[aX];
                series.push({
                    value: aI.value,
                    meta: aI.meta
                });

                if(tmp[bX] == null) {
                    let wta = whatToAppend(xAxis.slice(i + 1), aI, tmp);
                    wta.forEach(function(item) {
                        series.push(item);
                    });
                    i += wta.length;
                }

            } else if(aX != null && tmp[aX] != null) {
                let aI = tmp[aX];
                series.push({
                    value: aI.value,
                    meta: aI.meta
                });
            } else {
                series.push({
                    y: null
                });
            }

        }
        
        seriesMap[key] = series;
    }

    return [labels, seriesMap];
    
}

function whatToAppend(axis, start, tmp) {

    let startVal = start.value;
    let count = 0;
    let ret = [];
    let endVal = start.value;

    for(let i = 0; i < axis.length; i++) {
        count++;

        let x = axis[i];
        if(x == null) break;

        let item = tmp[x];
        if(item != null) {
            endVal = item.value;
            count -= 1; // Don't add this point
            break;
        }
    }

    let step = (endVal - startVal) / count;
    for(let i = 0; i < count; i++) {
        let x = axis[i];
        if(x == null) {
            ret.push({
                value: null
            });
            break;
        }
        ret.push({
            value: startVal + step * (i + 1),
            meta: ""
        });
    }

    return ret;

}

function getHours(t) {
    let now = (new Date()).getTime() / 1000;
    let diff = now - t;
    if(diff < 10800) {
        return Math.floor(diff / 60) + "m";
    } else {
        return Math.floor(diff / 3600) + "h";
    }
}

function isHidden(elem) {
    var style = window.getComputedStyle(elem);
    return (style.display === 'none');
}